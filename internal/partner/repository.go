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

// maxActiveShares is how many people one owner may share their wallets with at
// once. "Active" means a pending or joined outgoing link; rejected links do not
// count, so a declined invite frees up a slot.
const maxActiveShares = 5

// Invite creates (or re-opens) a pending share link from owner to partner.
//
// An owner may share their wallets with at most maxActiveShares people at once:
//   - if THIS person already has an active link -> ErrAlreadyLinked
//   - if the owner is already at the active limit -> ErrPartnerLimit
//   - otherwise re-open a previously rejected link to this person, or insert.
//
// Rejected links never count against the limit, so a declined invite leaves the
// owner free to invite someone else (or re-invite the same person).
func (r *Repository) Invite(ctx context.Context, ownerID, partnerID string) error {
	var activeCount, sameActive int
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE status IN ('pending', 'joined')),
			count(*) FILTER (WHERE status IN ('pending', 'joined') AND viewer_id = $2)
		FROM wallet_share_links
		WHERE owner_id = $1
	`, ownerID, partnerID).Scan(&activeCount, &sameActive)
	if err != nil {
		return err
	}
	if sameActive > 0 {
		return ErrAlreadyLinked
	}
	if activeCount >= maxActiveShares {
		return ErrPartnerLimit
	}

	// Below the limit. Re-open a prior rejected link to this person if one
	// exists; otherwise insert a fresh pending link.
	tag, err := r.pool.Exec(ctx,
		`UPDATE wallet_share_links SET status = 'pending', updated_at = now()
		 WHERE owner_id = $1 AND viewer_id = $2 AND status = 'rejected'`,
		ownerID, partnerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO wallet_share_links (owner_id, viewer_id, status) VALUES ($1, $2, 'pending')`,
		ownerID, partnerID,
	)
	return err
}

// Respond lets the invited viewer accept or reject. On accept, it fans out a
// joined viewer wallet_share for every wallet the owner currently holds.
func (r *Repository) Respond(ctx context.Context, linkID, partnerID, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID, invitedID, current string
	err = tx.QueryRow(ctx,
		`SELECT owner_id::text, viewer_id::text, status FROM wallet_share_links WHERE id = $1 FOR UPDATE`,
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
		`UPDATE wallet_share_links SET status = $2, updated_at = now() WHERE id = $1`,
		linkID, status,
	); err != nil {
		return err
	}

	if status == "joined" {
		if _, err = tx.Exec(ctx, `
			INSERT INTO wallet_shares (wallet_id, user_id, status, role, source)
			SELECT w.id, $2, 'joined', 'viewer', 'link'
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
// party is the viewer) and ones where they are the viewer (the other party is
// the owner).
func (r *Repository) List(ctx context.Context, userID string) ([]Link, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pl.id::text, 'owned' AS direction, pl.status,
		       u.id::text, u.email, u.name, pl.created_at, pl.updated_at
		FROM wallet_share_links pl
		JOIN users u ON u.id = pl.viewer_id
		WHERE pl.owner_id = $1
		UNION ALL
		SELECT pl.id::text, 'incoming' AS direction, pl.status,
		       u.id::text, u.email, u.name, pl.created_at, pl.updated_at
		FROM wallet_share_links pl
		JOIN users u ON u.id = pl.owner_id
		WHERE pl.viewer_id = $1
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
		`SELECT owner_id::text, viewer_id::text FROM wallet_share_links WHERE id = $1 FOR UPDATE`,
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
		WHERE user_id = $1 AND source = 'link'
		  AND wallet_id IN (SELECT id FROM wallets WHERE user_id = $2)
	`, partnerID, ownerID); err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `DELETE FROM wallet_share_links WHERE id = $1`, linkID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
