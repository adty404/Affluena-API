package activity

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeRepository struct {
	mu                sync.Mutex
	createdActivities []Activity
	listActivities    []Activity
	listTotal         int
	err               error
	createCalls       int
	createdCh         chan struct{}
	calledCh          chan struct{}
}

func (m *fakeRepository) Create(ctx context.Context, activity Activity) error {
	m.mu.Lock()
	m.createCalls++
	if m.err != nil {
		m.mu.Unlock()
		signalActivityTestChannel(m.calledCh)
		return m.err
	}
	m.createdActivities = append(m.createdActivities, activity)
	m.mu.Unlock()
	signalActivityTestChannel(m.calledCh)
	signalActivityTestChannel(m.createdCh)
	return nil
}

func (m *fakeRepository) List(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.listActivities, m.listTotal, nil
}

func (m *fakeRepository) waitForCreateCalls(t *testing.T, want int) {
	t.Helper()
	waitForActivityTestSignal(t, m.calledCh)

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createCalls != want {
		t.Fatalf("expected %d create calls, got %d", want, m.createCalls)
	}
}

func (m *fakeRepository) waitForCreatedActivities(t *testing.T, want int) []Activity {
	t.Helper()
	waitForActivityTestSignal(t, m.createdCh)

	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.createdActivities) != want {
		t.Fatalf("expected %d activity created, got %d", want, len(m.createdActivities))
	}
	activities := make([]Activity, len(m.createdActivities))
	copy(activities, m.createdActivities)
	return activities
}

func TestLogActivityExecutesSuccessfully(t *testing.T) {
	repo := &fakeRepository{createdCh: make(chan struct{}, 1), calledCh: make(chan struct{}, 1)}
	uc := NewUseCase(repo)

	entityID := "123"
	uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", &entityID, "Created wallet")

	activities := repo.waitForCreatedActivities(t, 1)
	act := activities[0]
	if act.UserID != "user-1" || act.ActionType != "CREATE" || act.EntityType != "WALLET" || *act.EntityID != "123" || act.Description != "Created wallet" {
		t.Fatalf("unexpected activity data: %+v", act)
	}
}

func TestLogActivityIgnoresRepositoryError(t *testing.T) {
	repo := &fakeRepository{err: errors.New("db error"), calledCh: make(chan struct{}, 1)}
	uc := NewUseCase(repo)

	uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", nil, "Created wallet")

	repo.waitForCreateCalls(t, 1)

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.createdActivities) != 0 {
		t.Fatalf("expected 0 activity created, got %d", len(repo.createdActivities))
	}
}

func TestLogActivityCopiesEntityIDBeforeBackgroundWrite(t *testing.T) {
	repo := &delayedEntityReadRepository{
		received: make(chan Activity, 1),
		release:  make(chan struct{}),
		done:     make(chan struct{}),
	}
	uc := NewUseCase(repo)

	entityID := "original"
	uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", &entityID, "Created wallet")

	<-repo.received
	entityID = "mutated"
	close(repo.release)
	waitForActivityTestSignal(t, repo.done)

	if repo.capturedEntityID == nil {
		t.Fatal("expected repository to receive entity id")
	}
	if *repo.capturedEntityID != "original" {
		t.Fatalf("expected copied entity id %q, got %q", "original", *repo.capturedEntityID)
	}
}

func TestListActivities(t *testing.T) {
	expected := []Activity{{ID: "act-1", UserID: "user-1"}}
	repo := &fakeRepository{listActivities: expected, listTotal: 1}
	uc := NewUseCase(repo)

	acts, total, err := uc.ListActivities(context.Background(), "user-1", 10, 0, "created_at_desc")
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

type delayedEntityReadRepository struct {
	received         chan Activity
	release          chan struct{}
	done             chan struct{}
	capturedEntityID *string
}

func (r *delayedEntityReadRepository) Create(ctx context.Context, activity Activity) error {
	r.received <- activity
	<-r.release
	if activity.EntityID != nil {
		entityID := *activity.EntityID
		r.capturedEntityID = &entityID
	}
	close(r.done)
	return nil
}

func (r *delayedEntityReadRepository) List(ctx context.Context, userID string, limit, offset int, sort string) ([]Activity, int, error) {
	return nil, 0, nil
}

func signalActivityTestChannel(ch chan struct{}) {
	if ch == nil {
		return
	}
	select {
	case ch <- struct{}{}:
	default:
	}
}

func waitForActivityTestSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for background activity logger")
	}
}

func TestLogActivityConcurrency(t *testing.T) {
	repo := &fakeRepository{calledCh: make(chan struct{}, 1000)}
	uc := NewUseCase(repo)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			uc.LogActivity(context.Background(), "user-1", "CREATE", "WALLET", nil, "Concurrency test")
		}(i)
	}

	wg.Wait()
}
