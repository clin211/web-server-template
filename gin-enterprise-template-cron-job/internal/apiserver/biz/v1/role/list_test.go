package role

import (
	"context"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	apistore "github.com/clin211/gin-enterprise-template/internal/apiserver/store"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"gorm.io/gorm"
)

type listRoleStore struct {
	total int64
	roles []*model.RoleM
}

func (s *listRoleStore) Create(context.Context, *model.RoleM) error {
	panic("unexpected call to Create")
}
func (s *listRoleStore) Update(context.Context, *model.RoleM) error {
	panic("unexpected call to Update")
}
func (s *listRoleStore) Delete(context.Context, *where.Options) error {
	panic("unexpected call to Delete")
}
func (s *listRoleStore) Get(context.Context, *where.Options) (*model.RoleM, error) {
	panic("unexpected call to Get")
}
func (s *listRoleStore) List(context.Context, *where.Options) (int64, []*model.RoleM, error) {
	return s.total, s.roles, nil
}
func (s *listRoleStore) GetByRoleCode(context.Context, string) (*model.RoleM, error) {
	panic("unexpected call to GetByRoleCode")
}
func (s *listRoleStore) AssignPermissions(context.Context, string, []string) error {
	panic("unexpected call to AssignPermissions")
}
func (s *listRoleStore) GetPermissions(context.Context, string) ([]*model.PermissionM, error) {
	panic("unexpected call to GetPermissions")
}
func (s *listRoleStore) RemovePermissions(context.Context, string) error {
	panic("unexpected call to RemovePermissions")
}

type roleBizStore struct {
	role apistore.RoleStore
}

func (s *roleBizStore) DB(context.Context, ...where.Where) *gorm.DB { return nil }
func (s *roleBizStore) TX(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (s *roleBizStore) User() apistore.UserStore                                     { return nil }
func (s *roleBizStore) Role() apistore.RoleStore                                     { return s.role }
func (s *roleBizStore) Permission() apistore.PermissionStore                         { return nil }
func (s *roleBizStore) Menu() apistore.MenuStore                                     { return nil }
func (s *roleBizStore) UserRole() apistore.UserRoleStore                             { return nil }
func (s *roleBizStore) ScheduledTask() apistore.ScheduledTaskStore                   { return nil }
func (s *roleBizStore) ScheduledTaskExecution() apistore.ScheduledTaskExecutionStore { return nil }

func TestList(t *testing.T) {
	tests := []struct {
		name          string
		pageSize      int32
		total         int64
		roles         []*model.RoleM
		wantPageToken bool
		wantTotal     int64
	}{
		{
			name:     "full page returns next token",
			pageSize: 2,
			total:    5,
			roles: []*model.RoleM{
				{ID: 9, RoleID: "role-9", RoleName: "role-9", RoleCode: "role-9"},
				{ID: 7, RoleID: "role-7", RoleName: "role-7", RoleCode: "role-7"},
			},
			wantPageToken: true,
			wantTotal:     5,
		},
		{
			name:     "short page does not return next token",
			pageSize: 3,
			total:    2,
			roles: []*model.RoleM{
				{ID: 5, RoleID: "role-5", RoleName: "role-5", RoleCode: "role-5"},
				{ID: 3, RoleID: "role-3", RoleName: "role-3", RoleCode: "role-3"},
			},
			wantPageToken: false,
			wantTotal:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			biz := New(&roleBizStore{role: &listRoleStore{total: tt.total, roles: tt.roles}}, nil)

			resp, err := biz.List(context.Background(), &v1.ListRoleRequest{PageSize: int64(tt.pageSize)})
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if resp.TotalCount != tt.wantTotal {
				t.Fatalf("TotalCount = %d, want %d", resp.TotalCount, tt.wantTotal)
			}
			if len(resp.Roles) != len(tt.roles) {
				t.Fatalf("len(Roles) = %d, want %d", len(resp.Roles), len(tt.roles))
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
			wantID := tt.roles[len(tt.roles)-1].ID
			if gotID != wantID {
				t.Fatalf("cursor id = %d, want %d", gotID, wantID)
			}
		})
	}
}
