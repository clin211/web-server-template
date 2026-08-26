package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/messaging/outbox"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"github.com/clin211/gin-enterprise-template/pkg/token"
)

func newTestTokenManager() *token.Manager {
	return token.NewManager("test-token-manager-key-with-sufficient-length", time.Hour, 24*time.Hour, token.WithIdentityKey("user_id"))
}

type fakeUserStore struct {
	created   *model.UserM
	get       func(context.Context, *where.Options) (*model.UserM, error)
	updatedID string
	updates   map[string]any
	deleted   bool
}

func (s *fakeUserStore) Create(_ context.Context, user *model.UserM) error {
	s.created = user
	return nil
}

func (s *fakeUserStore) Update(_ context.Context, userID string, updates map[string]any) error {
	s.updatedID = userID
	s.updates = updates
	return nil
}
func (s *fakeUserStore) Delete(context.Context, *where.Options) error {
	s.deleted = true
	return nil
}
func (s *fakeUserStore) Get(ctx context.Context, opts *where.Options) (*model.UserM, error) {
	if s.get != nil {
		return s.get(ctx, opts)
	}
	return nil, gorm.ErrRecordNotFound
}
func (*fakeUserStore) List(context.Context, *where.Options) (int64, []*model.UserM, error) {
	return 0, nil, nil
}
func (*fakeUserStore) UpdateLastLoginAt(context.Context, string, time.Time) error { return nil }

type fakeOutboxStore struct {
	calls int
}

type fakeAuditLogStore struct{ created []*model.AuditLogM }

type fakeUserLoginLogStore struct{ created []*model.UserLoginLogM }

func (s *fakeAuditLogStore) Create(_ context.Context, record *model.AuditLogM) error {
	s.created = append(s.created, record)
	return nil
}

func (s *fakeUserLoginLogStore) Create(_ context.Context, record *model.UserLoginLogM) error {
	s.created = append(s.created, record)
	return nil
}

func (s *fakeOutboxStore) Create(_ context.Context, event outbox.Event) error {
	s.calls++
	return nil
}
func (*fakeOutboxStore) Claim(context.Context, int, time.Duration) ([]outbox.Event, error) {
	return nil, nil
}
func (*fakeOutboxStore) MarkPublished(context.Context, string, int, time.Time) error { return nil }
func (*fakeOutboxStore) MarkFailed(context.Context, string, int, string, time.Time) error {
	return nil
}

type fakeStore struct {
	user      *fakeUserStore
	outbox    *fakeOutboxStore
	audit     *fakeAuditLogStore
	loginLog  *fakeUserLoginLogStore
	txCalled  bool
	txError   error
	txContext context.Context
}

var _ store.IStore = (*fakeStore)(nil)

func (*fakeStore) DB(context.Context, ...where.Where) *gorm.DB { return nil }
func (s *fakeStore) TX(ctx context.Context, fn func(context.Context) error) error {
	s.txCalled = true
	s.txContext = ctx
	s.txError = fn(ctx)
	return s.txError
}
func (s *fakeStore) User() store.UserStore           { return s.user }
func (s *fakeStore) Outbox() store.OutboxStore       { return s.outbox }
func (*fakeStore) Inbox() store.InboxStore           { return nil }
func (*fakeStore) DeadLetter() store.DeadLetterStore { return nil }
func (s *fakeStore) AuditLog() store.AuditLogStore {
	if s.audit == nil {
		s.audit = &fakeAuditLogStore{}
	}
	return s.audit
}
func (s *fakeStore) UserLoginLog() store.UserLoginLogStore {
	if s.loginLog == nil {
		s.loginLog = &fakeUserLoginLogStore{}
	}
	return s.loginLog
}

func TestCreateWritesUserWithoutOutbox(t *testing.T) {
	t.Parallel()

	datastore := &fakeStore{user: &fakeUserStore{}, outbox: &fakeOutboxStore{}}
	biz := New(datastore, newTestTokenManager())

	response, err := biz.Create(context.Background(), &v1.CreateUserRequest{
		Username: "alice",
		Password: "strong-password",
		Email:    "alice@example.com",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if datastore.txCalled {
		t.Fatal("Create() started an unnecessary transaction")
	}
	if datastore.user.created == nil || datastore.user.created.UserID != response.UserID {
		t.Fatalf("created user = %#v, response = %#v", datastore.user.created, response)
	}
	if datastore.outbox.calls != 0 {
		t.Fatalf("Outbox Create() calls = %d, want 0", datastore.outbox.calls)
	}
}

func TestUpdateAllowsPreviouslyUnusedEmail(t *testing.T) {
	t.Parallel()

	email := "new@example.com"
	call := 0
	userStore := &fakeUserStore{
		get: func(_ context.Context, _ *where.Options) (*model.UserM, error) {
			call++
			if call == 1 {
				return &model.UserM{UserID: "user-1", Username: "alice"}, nil
			}
			return nil, gorm.ErrRecordNotFound
		},
	}
	datastore := &fakeStore{user: userStore, outbox: &fakeOutboxStore{}}
	biz := New(datastore, newTestTokenManager())

	if _, err := biz.Update(context.Background(), &v1.UpdateUserRequest{UserID: "user-1", Email: &email}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got := userStore.updates["email"]; got != email {
		t.Fatalf("updated email = %#v, want %q", got, email)
	}
	if datastore.outbox.calls != 0 {
		t.Fatalf("Outbox Create() calls = %d, want 0", datastore.outbox.calls)
	}
}

func TestUserMutationsDoNotWriteOutbox(t *testing.T) {
	t.Parallel()

	password, err := authn.Encrypt("old-password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	tests := []struct {
		name            string
		changesPassword bool
		invoke          func(*userBiz) error
	}{
		{
			name: "delete",
			invoke: func(biz *userBiz) error {
				_, err := biz.Delete(context.Background(), &v1.DeleteUserRequest{UserID: "user-1"})
				return err
			},
		},
		{
			name:            "password changed",
			changesPassword: true,
			invoke: func(biz *userBiz) error {
				ctx := contextx.WithUserID(context.Background(), "user-1")
				_, err := biz.ChangePassword(ctx, &v1.ChangePasswordRequest{
					UserID: "user-1", OldPassword: "old-password", NewPassword: "new-password",
				})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userStore := &fakeUserStore{get: func(context.Context, *where.Options) (*model.UserM, error) {
				return &model.UserM{UserID: "user-1", Username: "alice", Password: password}, nil
			}}
			datastore := &fakeStore{user: userStore, outbox: &fakeOutboxStore{}}
			if err := tt.invoke(New(datastore, newTestTokenManager())); err != nil {
				t.Fatalf("mutation error = %v", err)
			}
			if !datastore.txCalled {
				t.Fatal("mutation did not start a transaction")
			}
			if datastore.outbox.calls != 0 {
				t.Fatalf("Outbox Create() calls = %d, want 0", datastore.outbox.calls)
			}
			if tt.changesPassword {
				if _, ok := userStore.updates["token_version"]; !ok {
					t.Fatal("password change did not increment token_version")
				}
			}
		})
	}
}

func TestLoginDoesNotRevealDisabledUser(t *testing.T) {
	t.Parallel()

	password, err := authn.Encrypt("strong-password")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	userStore := &fakeUserStore{get: func(context.Context, *where.Options) (*model.UserM, error) {
		return &model.UserM{UserID: "user-1", Username: "alice", Password: password, Status: 1}, nil
	}}
	biz := New(&fakeStore{user: userStore, outbox: &fakeOutboxStore{}}, newTestTokenManager())

	_, err = biz.Login(context.Background(), &v1.LoginRequest{Username: "alice", Password: "strong-password"})
	if !errors.Is(err, errno.ErrUnauthenticated) {
		t.Fatalf("Login() error = %v, want %v", err, errno.ErrUnauthenticated)
	}
}

func TestLoginUsesSameErrorAndAuditsUnknownUser(t *testing.T) {
	t.Parallel()

	datastore := &fakeStore{
		user: &fakeUserStore{get: func(context.Context, *where.Options) (*model.UserM, error) {
			return nil, gorm.ErrRecordNotFound
		}},
		outbox: &fakeOutboxStore{},
	}
	biz := New(datastore, newTestTokenManager())
	ctx := contextx.WithClientAddress(context.Background(), "192.0.2.10")
	ctx = contextx.WithUserAgent(ctx, "test-agent")

	_, err := biz.Login(ctx, &v1.LoginRequest{Username: "unknown", Password: "wrong-password"})
	if !errors.Is(err, errno.ErrUnauthenticated) {
		t.Fatalf("Login() error = %v, want generic invalid credentials", err)
	}
	if len(datastore.loginLog.created) != 1 {
		t.Fatalf("login audit count = %d, want 1", len(datastore.loginLog.created))
	}
	record := datastore.loginLog.created[0]
	if record.Status || record.IPAddress == nil || *record.IPAddress != "192.0.2.10" {
		t.Fatalf("login audit = %#v", record)
	}
}
