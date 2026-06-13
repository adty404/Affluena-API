package wallet

import (
	"context"
	"errors"

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
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor, goal_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, goal_id::text, created_at, updated_at
	`, userID, input.Name, input.Type, input.CurrencyCode, input.BalanceMinor, input.GoalID).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt,
	)
	return wallet, err
}

func (r *Repository) List(ctx context.Context, userID string, pagination page.Params) (page.Result[Wallet], error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT w.id::text, w.user_id::text, w.name, w.type, w.currency_code, w.balance_minor, w.goal_id::text, w.created_at, w.updated_at,
			CASE WHEN w.user_id = $1 THEN 'owner' ELSE 'member' END as role,
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
		if err := rows.Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt, &wallet.Role, &wallet.ShareStatus); err != nil {
			return page.Result[Wallet]{}, err
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
		SELECT DISTINCT w.id::text, w.user_id::text, w.name, w.type, w.currency_code, w.balance_minor, w.goal_id::text, w.created_at, w.updated_at,
			CASE WHEN w.user_id = $1 THEN 'owner' ELSE 'member' END as role,
			CASE WHEN w.user_id = $1 THEN 'joined' ELSE COALESCE(ws.status, 'pending') END as share_status
		FROM wallets w
		LEFT JOIN wallet_shares ws ON w.id = ws.wallet_id
		WHERE (w.user_id = $1 OR (ws.user_id = $1 AND ws.status IN ('pending', 'joined'))) AND w.id = $2
	`, userID, id).Scan(&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt, &wallet.Role, &wallet.ShareStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, err
	}

	wallet.Members, _ = r.GetMembers(ctx, wallet.ID)
	return wallet, nil
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateWalletInput) (Wallet, error) {
	var wallet Wallet
	err := r.pool.QueryRow(ctx, `
		UPDATE wallets
		SET name = $3, type = $4, currency_code = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, type, currency_code, balance_minor, goal_id::text, created_at, updated_at
	`, userID, id, input.Name, input.Type, input.CurrencyCode).Scan(
		&wallet.ID, &wallet.UserID, &wallet.Name, &wallet.Type, &wallet.CurrencyCode, &wallet.BalanceMinor, &wallet.GoalID, &wallet.CreatedAt, &wallet.UpdatedAt,
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

func (r *Repository) AddMember(ctx context.Context, walletID string, userID string, status string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO wallet_shares (wallet_id, user_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (wallet_id, user_id) DO UPDATE SET
			status = CASE
				WHEN wallet_shares.status = 'joined' THEN 'joined'
				ELSE EXCLUDED.status
			END,
			updated_at = now()
	`, walletID, userID, status)
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
		SELECT wallet_id::text, user_id::text, status, created_at, updated_at
		FROM wallet_shares
		WHERE wallet_id = $1
	`, walletID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []WalletMember
	for rows.Next() {
		var member WalletMember
		if err := rows.Scan(&member.WalletID, &member.UserID, &member.Status, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}
