package partner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakePartnerRepo struct {
	emailToID  map[string]string
	inviteErr  error
	respondErr error
	listResult []Link
	listErr    error
	revokeErr  error

	invitedOwner     string
	invitedPartner   string
	respondedLink    string
	respondedPartner string
	respondedStatus  string
	revokedLink      string
	revokedUser      string
}

func (f *fakePartnerRepo) FindUserByEmail(ctx context.Context, email string) (string, error) {
	if id, ok := f.emailToID[email]; ok {
		return id, nil
	}
	return "", ErrUserNotFound
}

func (f *fakePartnerRepo) Invite(ctx context.Context, ownerID, partnerID string) error {
	f.invitedOwner = ownerID
	f.invitedPartner = partnerID
	return f.inviteErr
}

func (f *fakePartnerRepo) Respond(ctx context.Context, linkID, partnerID, status string) error {
	f.respondedLink = linkID
	f.respondedPartner = partnerID
	f.respondedStatus = status
	return f.respondErr
}

func (f *fakePartnerRepo) List(ctx context.Context, userID string) ([]Link, error) {
	return f.listResult, f.listErr
}

func (f *fakePartnerRepo) Revoke(ctx context.Context, linkID, userID string) error {
	f.revokedLink = linkID
	f.revokedUser = userID
	return f.revokeErr
}

func TestInviteResolvesEmailAndDelegates(t *testing.T) {
	repo := &fakePartnerRepo{emailToID: map[string]string{"p@example.com": "partner-1"}}
	uc := NewUseCase(repo, nil)

	err := uc.Invite(context.Background(), "owner-1", InviteInput{Email: "p@example.com"})

	assert.NoError(t, err)
	assert.Equal(t, "owner-1", repo.invitedOwner)
	assert.Equal(t, "partner-1", repo.invitedPartner)
}

func TestInviteRejectsSelf(t *testing.T) {
	repo := &fakePartnerRepo{emailToID: map[string]string{"me@example.com": "owner-1"}}
	uc := NewUseCase(repo, nil)

	err := uc.Invite(context.Background(), "owner-1", InviteInput{Email: "me@example.com"})

	assert.ErrorIs(t, err, ErrSelfInvite)
	assert.Empty(t, repo.invitedPartner)
}

func TestInviteUnknownEmail(t *testing.T) {
	repo := &fakePartnerRepo{emailToID: map[string]string{}}
	uc := NewUseCase(repo, nil)

	err := uc.Invite(context.Background(), "owner-1", InviteInput{Email: "nobody@example.com"})

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestInvitePropagatesPartnerLimit(t *testing.T) {
	repo := &fakePartnerRepo{
		emailToID: map[string]string{"p@example.com": "partner-2"},
		inviteErr: ErrPartnerLimit,
	}
	uc := NewUseCase(repo, nil)

	err := uc.Invite(context.Background(), "owner-1", InviteInput{Email: "p@example.com"})

	assert.ErrorIs(t, err, ErrPartnerLimit)
}

func TestRespondDelegates(t *testing.T) {
	repo := &fakePartnerRepo{}
	uc := NewUseCase(repo, nil)

	err := uc.Respond(context.Background(), "partner-1", "link-1", RespondInput{Status: "joined"})

	assert.NoError(t, err)
	assert.Equal(t, "link-1", repo.respondedLink)
	assert.Equal(t, "partner-1", repo.respondedPartner)
	assert.Equal(t, "joined", repo.respondedStatus)
}

func TestRevokeDelegates(t *testing.T) {
	repo := &fakePartnerRepo{}
	uc := NewUseCase(repo, nil)

	err := uc.Revoke(context.Background(), "user-1", "link-1")

	assert.NoError(t, err)
	assert.Equal(t, "link-1", repo.revokedLink)
	assert.Equal(t, "user-1", repo.revokedUser)
}

func TestListReturnsLinks(t *testing.T) {
	repo := &fakePartnerRepo{listResult: []Link{{ID: "l1", Direction: DirectionOwned}}}
	uc := NewUseCase(repo, nil)

	links, err := uc.List(context.Background(), "user-1")

	assert.NoError(t, err)
	assert.Len(t, links, 1)
	assert.Equal(t, "l1", links[0].ID)
}
