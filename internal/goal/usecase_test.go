package goal

import (
	"context"
	"errors"
	"testing"
)

type fakeGoalRepository struct {
	createdInput CreateGoalInput
	createdUser  string
	createdGoal  Goal

	listed []Goal
	got    Goal
	getErr error

	findUserID string
	findErr    error
	addedGoal  string
	addedUser  string
	addedState string

	respondGoal   string
	respondUser   string
	respondStatus string
	respondErr    error
}

func (f *fakeGoalRepository) CreateWithOwnerWallet(ctx context.Context, userID string, input CreateGoalInput) (Goal, error) {
	f.createdUser = userID
	f.createdInput = input
	return f.createdGoal, nil
}

func (f *fakeGoalRepository) List(ctx context.Context, userID string) ([]Goal, error) {
	return f.listed, nil
}

func (f *fakeGoalRepository) Get(ctx context.Context, userID string, id string) (Goal, error) {
	if f.getErr != nil {
		return Goal{}, f.getErr
	}
	return f.got, nil
}

func (f *fakeGoalRepository) AddMember(ctx context.Context, goalID string, userID string, status string) error {
	f.addedGoal = goalID
	f.addedUser = userID
	f.addedState = status
	return nil
}

func (f *fakeGoalRepository) FindUserByEmail(ctx context.Context, email string) (string, error) {
	if f.findErr != nil {
		return "", f.findErr
	}
	return f.findUserID, nil
}

func (f *fakeGoalRepository) RespondInvite(ctx context.Context, goalID string, userID string, status string) error {
	f.respondGoal = goalID
	f.respondUser = userID
	f.respondStatus = status
	return f.respondErr
}

func TestGoalUsecaseCreateUsesAtomicRepositoryWorkflow(t *testing.T) {
	repo := &fakeGoalRepository{createdGoal: Goal{ID: "goal-1", Name: "Wedding"}}
	uc := NewUsecase(repo)

	created, err := uc.Create(context.Background(), "user-1", CreateGoalInput{Name: "Wedding", TargetAmountMinor: 1000})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID != "goal-1" {
		t.Fatalf("expected created goal, got %+v", created)
	}
	if repo.createdUser != "user-1" || repo.createdInput.Name != "Wedding" || repo.createdInput.TargetAmountMinor != 1000 {
		t.Fatalf("unexpected repository create call user=%q input=%+v", repo.createdUser, repo.createdInput)
	}
}

func TestGoalUsecaseInviteMemberRules(t *testing.T) {
	t.Run("owner can invite another existing user", func(t *testing.T) {
		repo := &fakeGoalRepository{
			got:        Goal{ID: "goal-1", UserID: "owner-1"},
			findUserID: "member-1",
		}
		uc := NewUsecase(repo)

		if err := uc.InviteMember(context.Background(), "owner-1", "goal-1", InviteMemberInput{Email: "member@example.test"}); err != nil {
			t.Fatalf("InviteMember returned error: %v", err)
		}
		if repo.addedGoal != "goal-1" || repo.addedUser != "member-1" || repo.addedState != "pending" {
			t.Fatalf("unexpected AddMember call goal=%q user=%q status=%q", repo.addedGoal, repo.addedUser, repo.addedState)
		}
	})

	t.Run("non-owner cannot invite", func(t *testing.T) {
		repo := &fakeGoalRepository{got: Goal{ID: "goal-1", UserID: "owner-1"}}
		uc := NewUsecase(repo)

		if err := uc.InviteMember(context.Background(), "member-1", "goal-1", InviteMemberInput{Email: "friend@example.test"}); !errors.Is(err, ErrNotAuthorized) {
			t.Fatalf("expected ErrNotAuthorized, got %v", err)
		}
		if repo.addedUser != "" {
			t.Fatalf("expected unauthorized invite to skip AddMember, got %q", repo.addedUser)
		}
	})

	t.Run("owner cannot invite self", func(t *testing.T) {
		repo := &fakeGoalRepository{
			got:        Goal{ID: "goal-1", UserID: "owner-1"},
			findUserID: "owner-1",
		}
		uc := NewUsecase(repo)

		if err := uc.InviteMember(context.Background(), "owner-1", "goal-1", InviteMemberInput{Email: "owner@example.test"}); err == nil {
			t.Fatal("expected self invite to fail")
		}
		if repo.addedUser != "" {
			t.Fatalf("expected self invite to skip AddMember, got %q", repo.addedUser)
		}
	})
}

func TestGoalUsecaseRespondInviteRequiresRouteMemberToMatchAuthenticatedUser(t *testing.T) {
	repo := &fakeGoalRepository{}
	uc := NewUsecase(repo)

	err := uc.RespondInvite(context.Background(), "member-1", "goal-1", "other-member", RespondInviteInput{Status: "joined"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected mismatched route member to be hidden as not found, got %v", err)
	}
	if repo.respondUser != "" {
		t.Fatalf("expected mismatched route member to skip repository, got %q", repo.respondUser)
	}

	if err := uc.RespondInvite(context.Background(), "member-1", "goal-1", "member-1", RespondInviteInput{Status: "joined"}); err != nil {
		t.Fatalf("RespondInvite returned error: %v", err)
	}
	if repo.respondGoal != "goal-1" || repo.respondUser != "member-1" || repo.respondStatus != "joined" {
		t.Fatalf("unexpected RespondInvite call goal=%q user=%q status=%q", repo.respondGoal, repo.respondUser, repo.respondStatus)
	}
}
