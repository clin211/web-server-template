package menu

import (
	"context"
	"fmt"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
)

// List 获取菜单列表（返回树形结构）.
func (b *menuBiz) List(ctx context.Context, rq *v1.ListMenuRequest) (*v1.ListMenuResponse, error) {
	total, menus, err := b.store.Menu().List(ctx, buildListMenuOptions(rq))
	if err != nil {
		return nil, fmt.Errorf("list menus: %w", err)
	}

	return &v1.ListMenuResponse{
		TotalCount: total,
		Menus:      buildMenuTree(menus),
		PageToken:  "",
	}, nil
}

// buildMenuTree 将扁平菜单列表转换为树形结构.
// 按 parent_id NULLS FIRST 和 sort_order ASC 排序，确保父菜单在子菜单之前.
func buildMenuTree(menus []*model.MenuM) []*v1.MenuTreeNode {
	if len(menus) == 0 {
		return nil
	}

	treeMap := make(map[string]*v1.MenuTreeNode, len(menus))
	roots := make([]*v1.MenuTreeNode, 0, len(menus))

	// 创建所有节点
	for _, menu := range menus {
		treeMap[menu.MenuID] = conversion.MenuModelToMenuTreeNodeV1(menu)
	}

	// 构建树关系
	for _, menu := range menus {
		node := treeMap[menu.MenuID]
		if menu.ParentID == nil || *menu.ParentID == "" {
			roots = append(roots, node)
		} else if parent, ok := treeMap[*menu.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// 孤儿节点（父节点不存在），作为根节点处理
			roots = append(roots, node)
		}
	}

	return roots
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