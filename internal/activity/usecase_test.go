package activity

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	createdActivities []Activity
	listActivities    []Activity
	listTotal         int
	err               error
}

func (m *fakeRepository) Create(ctx context.Context, activity Activity) error {
	if m.err != nil {
		return m.err
	}
	m.createdActivities = append(m.createdActivities, activity)
	return nil
}

func (m *fakeRepository) List(ctx context.Context, userID string, limit, offset int) ([]Activity, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.listActivities, m.listTotal, nil
}

func TestLogActivityExecutesSuccessfully(t *testing.T) {
	repo := &fakeRepository{}
	uc := NewUseCase(repo)

	entityID := "123"
	uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", &entityID, "Created wallet")

	// wait for goroutine
	time.Sleep(100 * time.Millisecond)

	if len(repo.createdActivities) != 1 {
		t.Fatalf("expected 1 activity created, got %d", len(repo.createdActivities))
	}
	act := repo.createdActivities[0]
	if act.UserID != "user-1" || act.ActionType != "CREATE" || act.EntityType != "WALLET" || *act.EntityID != "123" || act.Description != "Created wallet" {
		t.Fatalf("unexpected activity data: %+v", act)
	}
}

func TestLogActivityIgnoresRepositoryError(t *testing.T) {
	repo := &fakeRepository{err: errors.New("db error")}
	uc := NewUseCase(repo)

	uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", nil, "Created wallet")

	// wait for goroutine
	time.Sleep(100 * time.Millisecond)

	if len(repo.createdActivities) != 0 {
		t.Fatalf("expected 0 activity created, got %d", len(repo.createdActivities))
	}
}

func TestListActivities(t *testing.T) {
	expected := []Activity{{ID: "act-1", UserID: "user-1"}}
	repo := &fakeRepository{listActivities: expected, listTotal: 1}
	uc := NewUseCase(repo)

	acts, total, err := uc.ListActivities(context.Background(), "user-1", 10, 0)
	if err != nil {
		t.Fatalf("ListActivities returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(acts) != 1 || acts[0].ID != "act-1" {
		t.Fatalf("unexpected activities: %+v", acts)
	}
}
