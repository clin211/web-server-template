package user

import (
	"context"
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	apistore "github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"
)

type listUserStore struct {
	total int64
	users []*model.UserM
}

func (s *listUserStore) Create(context.Context, *model.UserM) error {
	panic("unexpected call to Create")
}
func (s *listUserStore) Update(context.Context, *model.UserM) error {
	panic("unexpected call to Update")
}
func (s *listUserStore) Delete(context.Context, *where.Options) error {
	panic("unexpected call to Delete")
}
func (s *listUserStore) Get(context.Context, *where.Options) (*model.UserM, error) {
	panic("unexpected call to Get")
}
func (s *listUserStore) List(context.Context, *where.Options) (int64, []*model.UserM, error) {
	return s.total, s.users, nil
}
func (s *listUserStore) UpdateLastLoginAt(context.Context, string, time.Time) error {
	panic("unexpected call to UpdateLastLoginAt")
}

type userBizStore struct {
	user apistore.UserStore
}

func (s *userBizStore) DB(context.Context, ...where.Where) *gorm.DB { return nil }
func (s *userBizStore) TX(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (s *userBizStore) User() apistore.UserStore                                     { return s.user }
func (s *userBizStore) Role() apistore.RoleStore                                     { return nil }
func (s *userBizStore) Permission() apistore.PermissionStore                         { return nil }
func (s *userBizStore) Menu() apistore.MenuStore                                      { return nil }
func (s *userBizStore) MenuRole() apistore.MenuRoleStore                              { return nil }
func (s *userBizStore) UserRole() apistore.UserRoleStore                              { return nil }
func (s *userBizStore) ScheduledTask() apistore.ScheduledTaskStore                   { return nil }
func (s *userBizStore) ScheduledTaskExecution() apistore.ScheduledTaskExecutionStore { return nil }

func TestList(t *testing.T) {
	tests := []struct {
		name          string
		pageSize      int32
		total         int64
		users         []*model.UserM
		wantPageToken bool
		wantTotal     int64
	}{
		{
			name:     "full page preserves order and returns token",
			pageSize: 2,
			total:    4,
			users: []*model.UserM{
				{ID: 11, UserID: "user-11", Username: "alice", Nickname: "Alice"},
				{ID: 7, UserID: "user-7", Username: "bob", Nickname: "Bob"},
			},
			wantPageToken: true,
			wantTotal:     4,
		},
		{
			name:     "short page preserves order and omits token",
			pageSize: 3,
			total:    2,
			users: []*model.UserM{
				{ID: 5, UserID: "user-5", Username: "carol", Nickname: "Carol"},
				{ID: 3, UserID: "user-3", Username: "dave", Nickname: "Dave"},
			},
			wantPageToken: false,
			wantTotal:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := contextx.WithUsername(context.Background(), known.AdminUsername)
			biz := New(&userBizStore{user: &listUserStore{total: tt.total, users: tt.users}}, nil)

			resp, err := biz.List(ctx, &v1.ListUserRequest{PageSize: int64(tt.pageSize)})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if resp.TotalCount != tt.wantTotal {
				t.Fatalf("TotalCount = %d, want %d", resp.TotalCount, tt.wantTotal)
			}
			if len(resp.Users) != len(tt.users) {
				t.Fatalf("len(Users) = %d, want %d", len(resp.Users), len(tt.users))
			}

			for i, wantUser := range tt.users {
				if resp.Users[i].UserID != wantUser.UserID {
					t.Fatalf("Users[%d].UserID = %q, want %q", i, resp.Users[i].UserID, wantUser.UserID)
				}
				if resp.Users[i].Username != wantUser.Username {
					t.Fatalf("Users[%d].Username = %q, want %q", i, resp.Users[i].Username, wantUser.Username)
				}
			}

			if !tt.wantPageToken {
				if resp.PageToken != "" {
					t.Fatalf("PageToken = %q, want empty", resp.PageToken)
				}
				return
			}

			if resp.PageToken == "" {
				t.Fatal("PageToken is empty, want non-empty")
			}

			cursor, err := pagination.DecodeCursor(resp.PageToken)
			if err != nil {
				t.Fatalf("DecodeCursor() error = %v", err)
			}
			gotID, ok := cursor.GetInt64("id")
			if !ok {
				t.Fatal("cursor.GetInt64(id) returned false")
			}
			wantID := tt.users[len(tt.users)-1].ID
			if gotID != wantID {
				t.Fatalf("cursor id = %d, want %d", gotID, wantID)
			}
		})
	}
}
