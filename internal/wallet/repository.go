package wallet

import (
	"context"
	"errors"
	"time"

	"affluena-api/internal/page"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, input CreateWalletInput) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor, color, description, goal_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, color, description, goal_id::text, created_at, updated_at
	`, userID, input.Name, input.Type, input.CurrencyCode, input.BalanceMinor, input.Color, input.Description, input.GoalID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.Color, &wallet.Description, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	return wallet, err
}

func (r *Repository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT w.id::text, w.user_id::text, w.name, w.type, w.currency_code, w.balance_minor, w.color, w.description, w.goal_id::text, w.created_at, w.updated_at,
			CASE WHEN w.user_id = $1 THEN 'owner' ELSE COALESCE(ws.role, 'member') END as role,
			CASE WHEN w.user_id = $1 THEN 'joined' ELSE ws.status END as share_status
		FROM wallets w
		LEFT JOIN wallet_shares ws ON w.id = ws.wallet_id
		WHERE w.user_id = $1 OR (ws.user_id = $1 AND ws.status IN ('pending', 'joined'))
		ORDER BY `+walletOrderBy(pagination.Sort)+`
		LIMIT $2 OFFSET $3
	`, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return page.Result[Wallet]{}, err
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var wallet Wallet
		if err := rows.Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.Color, &wallet.Description, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt, &wallet.Role, &wallet.ShareStatus); err != nil {
			return page.Result[Wallet]{}, err
		}
		// A pending (not-yet-accepted) invitee may see that the wallet exists so
		// they can accept the invite, but not its balance until they join.
		if wallet.Role != "owner" && wallet.ShareStatus != "joined" {
			wallet.BalanceMinor = 0
		}
		wallets = append(wallets, wallet)
	}
	if err := rows.Err(); err != nil {
		return page.Result[Wallet]{}, err
	}
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT w.id)
		FROM wallets w
		LEFT JOIN wallet_shares ws ON w.id = ws.wallet_id
		WHERE w.user_id = $1 OR (ws.user_id = $1 AND ws.status IN ('pending', 'joined'))
	`, userID).Scan(&total); err != nil {
		return page.Result[Wallet]{}, err
	}
	return page.NewResult(wallets, pagination, total), nil
}

func walletOrderBy(sort string) string {
	switch sort {
	case "created_at_asc":
		return "created_at ASC"
	case "name_asc":
		return "name ASC"
	case "name_desc":
		return "name DESC"
	case "balance_desc":
		return "balance_minor DESC, created_at DESC"
	case "balance_asc":
		return "balance_minor ASC, created_at DESC"
	default:
		return "created_at DESC"
	}
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		SELECT DISTINCT w.id::text, w.user_id::text, w.name, w.type, w.currency_code, w.balance_minor, w.color, w.description, w.goal_id::text, w.created_at, w.updated_at,
			CASE WHEN w.user_id = $1 THEN 'owner' ELSE COALESCE(ws.role, 'member') END as role,
			CASE WHEN w.user_id = $1 THEN 'joined' ELSE COALESCE(ws.status, 'pending') END as share_status
		FROM wallets w
		LEFT JOIN wallet_shares ws ON w.id = ws.wallet_id
		WHERE (w.user_id = $1 OR (ws.user_id = $1 AND ws.status IN ('pending', 'joined'))) AND w.id = $2
	`, userID, id).Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.Color, &wallet.Description, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt, &wallet.Role, &wallet.ShareStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, err
	}

	// Owner and joined members get the full wallet (balance + roster). A pending
	// invitee can see the wallet to accept the invite, but not its balance or the
	// member roster (which would leak other members' emails) until they join.
	if wallet.Role == "owner" || wallet.ShareStatus == "joined" {
		wallet.Members, _ = r.GetMembers(ctx, wallet.ID)
	} else {
		wallet.BalanceMinor = 0
	}
	return wallet, nil
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		UPDATE wallets
		SET name = $3, type = $4, currency_code = $5, color = $6, description = $7, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, color, description, goal_id::text, created_at, updated_at
	`, userID, id, input.Name, input.Type, input.CurrencyCode, input.Color, input.Description).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.Color, &wallet.Description, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	return wallet, err
}

func (r *Repository) Delete(ctx context.Context, userID string, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM wallets WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, email).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("user not found")
	}
	return id, err
}

func (r *Repository) AddMember(ctx context.Context, walletID string, userID string, status string, role string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wallet_shares (wallet_id, user_id, status, role)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (wallet_id, user_id) DO UPDATE SET
			status = CASE
				WHEN wallet_shares.status = 'joined' THEN 'joined'
				ELSE EXCLUDED.status
			END,
			role = EXCLUDED.role,
			updated_at = now()
	`, walletID, userID, status, role)
	return err
}

// AddPartnerViewerShares grants every joined account-level partner of the owner
// a viewer share on the given wallet (used when a new wallet is created so the
// "partner sees all my wallets" invariant holds). Tagged source='partner' so it
// is cleaned up if the partner link is later revoked; never clobbers a manual
// share.
func (r *Repository) AddPartnerViewerShares(ctx context.Context, ownerID string, walletID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wallet_shares (wallet_id, user_id, status, role, source)
		SELECT $1, pl.partner_id, 'joined', 'viewer', 'partner'
		FROM partner_links pl
		WHERE pl.owner_id = $2 AND pl.status = 'joined'
		ON CONFLICT (wallet_id, user_id) DO NOTHING
	`, walletID, ownerID)
	return err
}

func (r *Repository) RespondInvite(ctx context.Context, walletID string, userID string, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID string
	if err := tx.QueryRow(ctx, `
		SELECT user_id::text
		FROM wallets
		WHERE id = $1
		FOR UPDATE
	`, walletID).Scan(&ownerID); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if ownerID == userID {
		return ErrOwnerCannotRespond
	}

	var currentStatus string
	if err := tx.QueryRow(ctx, `
		SELECT status
		FROM wallet_shares
		WHERE wallet_id = $1 AND user_id = $2
		FOR UPDATE
	`, walletID, userID).Scan(&currentStatus); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if currentStatus == "joined" && status == "joined" {
		return ErrInviteAlreadyJoined
	}
	if currentStatus == "rejected" {
		return ErrInviteAlreadyRejected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE wallet_shares
		SET status = $3, updated_at = now()
		WHERE wallet_id = $1 AND user_id = $2
	`, walletID, userID, status); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetMembers(ctx context.Context, walletID string) ([]WalletMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT wallet_id::text, user_id::text, email, role, status, created_at, updated_at FROM (
			SELECT w.id AS wallet_id, w.user_id, u.email, 'owner' AS role, 'joined' AS status, w.created_at, w.updated_at
			FROM wallets w
			JOIN users u ON u.id = w.user_id
			WHERE w.id = $1
			UNION ALL
			SELECT ws.wallet_id, ws.user_id, u.email, ws.role, ws.status, ws.created_at, ws.updated_at
			FROM wallet_shares ws
			JOIN users u ON u.id = ws.user_id
			WHERE ws.wallet_id = $1
		) m
		ORDER BY role DESC, status ASC, created_at ASC
	`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []WalletMember
	for rows.Next() {
		var member WalletMember
		if err := rows.Scan(&member.WalletID, &member.UserID, &member.Email, &member.Role, &member.Status, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *Repository) GetAnalytics(ctx context.Context, userID string, walletID string, month string) (WalletAnalytics, error) {
	var analytics WalletAnalytics
	analytics.WalletID = walletID
	analytics.Month = month

	// access check first
	var accessible bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM wallets w
			WHERE w.id = $1 AND w.user_id = $2
		) OR EXISTS(
			SELECT 1 FROM wallet_shares ws
			WHERE ws.wallet_id = $1 AND ws.user_id = $2 AND ws.status = 'joined'
		)
	`, walletID, userID).Scan(&accessible)
	if err != nil {
		return analytics, err
	}
	if !accessible {
		return analytics, ErrNotFound
	}

	// aggregate monthly transactions
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount_minor ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_minor ELSE 0 END), 0),
			COUNT(*),
			MAX(transaction_at)
		FROM transactions
		WHERE wallet_id = $1 AND to_char(transaction_at, 'YYYY-MM') = $2
	`, walletID, month).Scan(&analytics.InflowMinor, &analytics.OutflowMinor, &analytics.TransactionCount, &analytics.LastActivityAt)
	if err != nil {
		return analytics, err
	}
	// also include transfers into wallet as inflow
	err = r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount_minor), 0),
			COUNT(*),
			MAX(transaction_at)
		FROM transactions
		WHERE to_wallet_id = $1 AND type = 'transfer' AND to_char(transaction_at, 'YYYY-MM') = $2
	`, walletID, month).Scan(&analytics.InflowMinor, &analytics.TransactionCount, &analytics.LastActivityAt)
	// ignore err; transfers optional

	// last activity overall if not in month
	if analytics.LastActivityAt == nil {
		var lastActivity time.Time
		err = r.pool.QueryRow(ctx, `
			SELECT MAX(transaction_at) FROM transactions WHERE wallet_id = $1 OR to_wallet_id = $1
		`, walletID).Scan(&lastActivity)
		if err == nil && !lastActivity.IsZero() {
			analytics.LastActivityAt = &lastActivity
		}
	}

	return analytics, nil
}

// GetAccessLevel returns user's access level for a wallet.
func (r *Repository) GetAccessLevel(ctx context.Context, userID string, walletID string) (AccessLevel, error) {
	var isOwner bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM wallets WHERE id = $1 AND user_id = $2
		)
	`, walletID, userID).Scan(&isOwner)
	if err != nil {
		return AccessNone, err
	}
	if isOwner {
		return AccessOwner, nil
	}

	var role string
	err = r.pool.QueryRow(ctx, `
		SELECT role FROM wallet_shares WHERE wallet_id = $1 AND user_id = $2 AND status = 'joined'
	`, walletID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return AccessNone, nil
	}
	if err != nil {
		return AccessNone, err
	}
	if role == "viewer" {
		return AccessViewer, nil
	}
	return AccessMember, nil
}
