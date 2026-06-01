package menu

import (
	"context"
	"fmt"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// SortMenu 批量更新菜单排序.
func (b *menuBiz) SortMenu(ctx context.Context, rq *v1.SortMenuRequest) (*v1.SortMenuResponse, error) {
	// 构建菜单ID到排序值的映射
	items := make(map[string]int32, len(rq.Items))
	for _, item := range rq.Items {
		items[item.GetMenuID()] = item.GetSortOrder()
	}

	// 批量更新排序，使用单条 SQL
	if err := b.store.Menu().BatchUpdateSortOrder(ctx, items); err != nil {
		return nil, fmt.Errorf("batch update sort order: %w", err)
	}

	return &v1.SortMenuResponse{}, nil
}
