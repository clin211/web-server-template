package role

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// List 获取角色列表.
func (b *roleBiz) List(ctx context.Context, rq *v1.ListRoleRequest) (*v1.ListRoleResponse, error) {
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())
	total, roles, err := b.store.Role().List(ctx, buildListRoleOptions(rq))
	if err != nil {
		return nil, err
	}

	nextPageToken := pagination.NextPageToken(len(roles), pageSize, func() int64 {
		return roles[len(roles)-1].ID
	})

	return &v1.ListRoleResponse{
		TotalCount: total,
		Roles:      conversion.RoleModelListToRoleV1List(roles),
		PageToken:  nextPageToken,
	}, nil
}

// buildListRoleOptions 构建角色列表查询选项.
func buildListRoleOptions(rq *v1.ListRoleRequest) *where.Options {
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())

	opts := where.NewWhere(where.WithLimit(int64(pageSize)))

	// 解析 page_token 获取游标
	pageToken := rq.GetPageToken()
	if pageToken != "" {
		decodedCursor, err := pagination.DecodeCursor(pageToken)
		if err == nil {
			if id, ok := decodedCursor.GetInt64("id"); ok {
				opts.Cursor = &id
			}
		}
	}

	if rq.GetStatus() != 0 {
		opts.F("status", rq.GetStatus())
	}

	if rq.GetKeyword() != "" {
		keyword := "%" + rq.GetKeyword() + "%"
		opts.Q("(role_name LIKE ? OR role_code LIKE ?)", keyword, keyword)
	}

	return opts
}
