package menu

import (
	"context"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// List 获取菜单列表.
func (b *menuBiz) List(ctx context.Context, rq *v1.ListMenuRequest) (*v1.ListMenuResponse, error) {
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())
	total, menus, err := b.store.Menu().List(ctx, buildListMenuOptions(rq))
	if err != nil {
		return nil, err
	}

	nextPageToken := pagination.NextPageToken(len(menus), pageSize, func() int64 {
		return menus[len(menus)-1].ID
	})

	return &v1.ListMenuResponse{
		TotalCount: total,
		Menus:      conversion.MenuModelListToMenuV1List(menus),
		PageToken:  nextPageToken,
	}, nil
}

// buildListMenuOptions 构建菜单列表查询选项.
func buildListMenuOptions(rq *v1.ListMenuRequest) *where.Options {
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

	if rq.GetMenuType() != "" {
		opts.F("menu_type", rq.GetMenuType())
	}

	if rq.GetParentID() != "" {
		opts.F("parent_id", rq.GetParentID())
	}

	return opts
}
