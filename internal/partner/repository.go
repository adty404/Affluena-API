package partner

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id::text FROM users WHERE email = $1`, email).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return id, err
}

// Invite creates (or re-opens) a pending partner link from owner to partner.
// A pending or joined link is treated as a conflict; a previously rejected one
// is re-opened to pending.
func (r *Repository) Invite(ctx context.Context, ownerID, partnerID string) error {
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT status FROM partner_links WHERE owner_id = $1 AND partner_id = $2`,
		ownerID, partnerID,
	).Scan(&status)
	if err == nil {
		if status == "joined" || status == "pending" {
			return ErrAlreadyLinked
		}
		_, err = r.pool.Exec(ctx,
			`UPDATE partner_links SET status = 'pending', updated_at = now() WHERE owner_id = $1 AND partner_id = $2`,
			ownerID, partnerID,
		)
		return err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO partner_links (owner_id, partner_id, status) VALUES ($1, $2, 'pending')`,
		ownerID, partnerID,
	)
	return err
}

// Respond lets the invited partner accept or reject. On accept, it fans out a
// joined viewer wallet_share for every wallet the owner currently holds.
func (r *Repository) Respond(ctx context.Context, linkID, partnerID, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID, invitedID, current string
	err = tx.QueryRow(ctx,
		`SELECT owner_id::text, partner_id::text, status FROM partner_links WHERE id = $1 FOR UPDATE`,
		linkID,
	).Scan(&ownerID, &invitedID, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if invitedID != partnerID {
		return ErrNotAuthorized
	}

	if _, err = tx.Exec(ctx,
		`UPDATE partner_links SET status = $2, updated_at = now() WHERE id = $1`,
		linkID, status,
	); err != nil {
		return err
	}

	if status == "joined" {
		if _, err = tx.Exec(ctx, `
			INSERT INTO wallet_shares (wallet_id, user_id, status, role, source)
			SELECT w.id, $2, 'joined', 'viewer', 'partner'
			FROM wallets w
			WHERE w.user_id = $1
			ON CONFLICT (wallet_id, user_id) DO NOTHING
		`, ownerID, partnerID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// List returns the caller's links in both directions: ones they own (the other
// party is the partner) and ones where they are the partner (the other party is
// the owner).
func (r *Repository) List(ctx context.Context, userID string) ([]Link, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pl.id::text, 'owned' AS direction, pl.status,
		       u.id::text, u.email, u.name, pl.created_at, pl.updated_at
		FROM partner_links pl
		JOIN users u ON u.id = pl.partner_id
		WHERE pl.owner_id = $1
		UNION ALL
		SELECT pl.id::text, 'incoming' AS direction, pl.status,
		       u.id::text, u.email, u.name, pl.created_at, pl.updated_at
		FROM partner_links pl
		JOIN users u ON u.id = pl.owner_id
		WHERE pl.partner_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.Direction, &l.Status, &l.UserID, &l.Email, &l.Name, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// Revoke deletes the link (either party may do so) and cleans up the viewer
// shares it auto-created, leaving any manual shares intact.
func (r *Repository) Revoke(ctx context.Context, linkID, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID, partnerID string
	err = tx.QueryRow(ctx,
		`SELECT owner_id::text, partner_id::text FROM partner_links WHERE id = $1 FOR UPDATE`,
		linkID,
	).Scan(&ownerID, &partnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if userID != ownerID && userID != partnerID {
		return ErrNotAuthorized
	}

	if _, err = tx.Exec(ctx, `
		DELETE FROM wallet_shares
		WHERE user_id = $1 AND source = 'partner'
		  AND wallet_id IN (SELECT id FROM wallets WHERE user_id = $2)
	`, partnerID, ownerID); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `DELETE FROM partner_links WHERE id = $1`, linkID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
