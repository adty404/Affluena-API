package goal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	var goal Goal
	err := r.pool.QueryRow(ctx, `
		INSERT INTO goals (user_id, name, target_amount_minor, deadline, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id::text, user_id::text, name, target_amount_minor, deadline, status, created_at, updated_at
	`, userID, input.Name, input.TargetAmountMinor, input.Deadline).Scan(
		&goal.ID, &goal.UserID, &goal.Name, &goal.TargetAmountMinor, &goal.Deadline, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt,
	)
	return goal, err
}

func (r *Repository) CreateWithOwnerWallet(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Goal{}, err
	}
	defer tx.Rollback(ctx)

	goal, err := createGoal(ctx, tx, userID, input)
	if err != nil {
		return Goal{}, err
	}
	if err := addMember(ctx, tx, goal.ID, userID, "joined"); err != nil {
		return Goal{}, err
	}
	if err := createGoalWallet(ctx, tx, userID, goal.ID, goal.Name); err != nil {
		return Goal{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Goal{}, err
	}
	return r.Get(ctx, userID, goal.ID)
}

func (r *Repository) Update(ctx context.Context, userID string, id string, input UpdateGoalInput) (Goal, error) {
	var goal Goal
	var status *string
	if input.Status != "" {
		status = &input.Status
	}
	err := r.pool.QueryRow(ctx, `
		UPDATE goals
		SET name = $3, target_amount_minor = $4, deadline = $5,
		    status = COALESCE($6, status), updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING id::text, user_id::text, name, target_amount_minor, deadline, status, created_at, updated_at
	`, userID, id, input.Name, input.TargetAmountMinor, input.Deadline, status).Scan(
		&goal.ID, &goal.UserID, &goal.Name, &goal.TargetAmountMinor, &goal.Deadline, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Goal{}, ErrNotFound
	}
	if err != nil {
		return Goal{}, err
	}
	goal.CollectedAmountMinor, _ = r.getCollectedAmount(ctx, goal.ID)
	goal.Members, _ = r.GetMembers(ctx, goal.ID)
	return goal, nil
}

func (r *Repository) List(ctx context.Context, userID string) ([]Goal, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT g.id::text, g.user_id::text, g.name, g.target_amount_minor, g.deadline, g.status, g.created_at, g.updated_at,
			CASE WHEN g.user_id = $1 THEN 'joined' ELSE gm.status END AS member_status
		FROM goals g
		LEFT JOIN goal_members gm ON g.id = gm.goal_id
		WHERE g.user_id = $1 OR (gm.user_id = $1 AND gm.status IN ('pending', 'joined'))
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []Goal
	for rows.Next() {
		var goal Goal
		var memberStatus string
		if err := rows.Scan(&goal.ID, &goal.UserID, &goal.Name, &goal.TargetAmountMinor, &goal.Deadline, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt, &memberStatus); err != nil {
			return nil, err
		}
		// Pending invitees see the goal (to accept it) but not the collective
		// pool balance until they join.
		if goal.UserID == userID || memberStatus == "joined" {
			goal.CollectedAmountMinor, _ = r.getCollectedAmount(ctx, goal.ID)
		}
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

func (r *Repository) Get(ctx context.Context, userID string, id string) (Goal, error) {
	var goal Goal
	var memberStatus string
	err := r.pool.QueryRow(ctx, `
		SELECT DISTINCT g.id::text, g.user_id::text, g.name, g.target_amount_minor, g.deadline, g.status, g.created_at, g.updated_at,
			CASE WHEN g.user_id = $1 THEN 'joined' ELSE COALESCE(gm.status, 'pending') END AS member_status
		FROM goals g
		LEFT JOIN goal_members gm ON g.id = gm.goal_id
		WHERE (g.user_id = $1 OR (gm.user_id = $1 AND gm.status IN ('pending', 'joined'))) AND g.id = $2
	`, userID, id).Scan(&goal.ID, &goal.UserID, &goal.Name, &goal.TargetAmountMinor, &goal.Deadline, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt, &memberStatus)

	if errors.Is(err, pgx.ErrNoRows) {
		return Goal{}, ErrNotFound
	}
	if err != nil {
		return Goal{}, err
	}

	// Pending (not-yet-accepted) invitees can see the goal to accept it, but not
	// the collective pool balance or the member roster until they join.
	if goal.UserID == userID || memberStatus == "joined" {
		goal.CollectedAmountMinor, _ = r.getCollectedAmount(ctx, goal.ID)
		goal.Members, _ = r.GetMembers(ctx, goal.ID)
	}
	return goal, nil
}

func (r *Repository) getCollectedAmount(ctx context.Context, goalID string) (int64, error) {
	var total *int64
	err := r.pool.QueryRow(ctx, `SELECT SUM(balance_minor) FROM wallets WHERE goal_id = $1`, goalID).Scan(&total)
	if err != nil || total == nil {
		return 0, err
	}
	return *total, nil
}

func (r *Repository) GetMembers(ctx context.Context, goalID string) ([]GoalMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT goal_id::text, user_id::text, status, created_at, updated_at
		FROM goal_members
		WHERE goal_id = $1
	`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []GoalMember
	for rows.Next() {
		var member GoalMember
		if err := rows.Scan(&member.GoalID, &member.UserID, &member.Status, &member.CreatedAt, &member.UpdatedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (r *Repository) AddMember(ctx context.Context, goalID string, userID string, status string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO goal_members (goal_id, user_id, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (goal_id, user_id) DO UPDATE SET
			status = CASE
				WHEN goal_members.status = 'joined' THEN 'joined'
				ELSE EXCLUDED.status
			END,
			updated_at = now()
	`, goalID, userID, status)
	return err
}

func (r *Repository) RespondInvite(ctx context.Context, goalID string, userID string, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID string
	var goalName string
	if err := tx.QueryRow(ctx, `
		SELECT user_id::text, name
		FROM goals
		WHERE id = $1
		FOR UPDATE
	`, goalID).Scan(&ownerID, &goalName); errors.Is(err, pgx.ErrNoRows) {
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
		FROM goal_members
		WHERE goal_id = $1 AND user_id = $2
		FOR UPDATE
	`, goalID, userID).Scan(&currentStatus); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if currentStatus == "joined" {
		return ErrInviteAlreadyJoined
	}
	if currentStatus == "rejected" {
		return ErrInviteAlreadyRejected
	}

	if _, err := tx.Exec(ctx, `
		UPDATE goal_members
		SET status = $3, updated_at = now()
		WHERE goal_id = $1 AND user_id = $2
	`, goalID, userID, status); err != nil {
		return err
	}

	if status == "joined" {
		if err := createGoalWallet(ctx, tx, userID, goalID, goalName); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) UpdateMemberStatus(ctx context.Context, goalID string, userID string, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE goal_members
		SET status = $3, updated_at = now()
		WHERE goal_id = $1 AND user_id = $2
	`, goalID, userID, status)
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

func createGoal(ctx context.Context, tx pgx.Tx, userID string, input CreateGoalInput) (Goal, error) {
	var goal Goal
	err := tx.QueryRow(ctx, `
		INSERT INTO goals (user_id, name, target_amount_minor, deadline, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id::text, user_id::text, name, target_amount_minor, deadline, status, created_at, updated_at
	`, userID, input.Name, input.TargetAmountMinor, input.Deadline).Scan(
		&goal.ID, &goal.UserID, &goal.Name, &goal.TargetAmountMinor, &goal.Deadline, &goal.Status, &goal.CreatedAt, &goal.UpdatedAt,
	)
	return goal, err
}

func addMember(ctx context.Context, tx pgx.Tx, goalID string, userID string, status string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO goal_members (goal_id, user_id, status)
		VALUES ($1, $2, $3)
	`, goalID, userID, status)
	return err
}

func createGoalWallet(ctx context.Context, tx pgx.Tx, userID string, goalID string, goalName string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO wallets (user_id, name, type, currency_code, balance_minor, goal_id)
		VALUES ($1, $2, 'goal', 'IDR', 0, $3)
	`, userID, goalWalletName(goalName, goalID), goalID)
	return err
}

func goalWalletName(goalName string, goalID string) string {
	suffix := goalID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	return fmt.Sprintf("[Goal] %s (%s)", goalName, suffix)
}
