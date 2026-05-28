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

	// 生成下一页的 page_token
	// 只有当返回的数据量等于 pageSize 时，才说明可能有下一页
	var nextPageToken string
	if len(roles) > pageSize {
		lastRole := roles[len(roles)-1]
		cursor, err := pagination.NewCursor("id", lastRole.ID)
		if err == nil {
			nextPageToken, _ = cursor.Encode()
		}
	}

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
