package user

import (
	"context"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"

	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// List 实现 UserBiz 接口中的 List 方法.
func (b *userBiz) List(ctx context.Context, rq *v1.ListUserRequest) (*v1.ListUserResponse, error) {
	// 解析 page_token 获取游标
	cursor := decodeUserCursor(ctx, rq.GetPageToken())

	// 构建 where.Options，使用游标分页
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())

	whr := where.NewWhere(where.WithLimit(int64(pageSize + 1)))
	if cursor != nil {
		whr.Cursor = cursor
	}

	count, userList, err := b.store.User().List(ctx, whr)
	if err != nil {
		return nil, err
	}

	hasMore := len(userList) > pageSize
	if hasMore {
		userList = userList[:pageSize]
	}
	users := conversion.UserModelListToUserV1List(userList)

	nextPageToken := ""
	if hasMore {
		nextPageToken = pagination.NextPageToken(len(userList), pageSize, func() int64 { return userList[len(userList)-1].ID })
	}

	slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

	return &v1.ListUserResponse{
		TotalCount: count,
		Users:      users,
		PageToken:  nextPageToken,
	}, nil
}

func decodeUserCursor(ctx context.Context, token string) *int64 {
	if token == "" {
		return nil
	}
	cursor, err := pagination.DecodeCursor(token)
	if err != nil {
		// 非法或过期 token 沿用 API 兼容策略，回到第一页。
		slog.WarnContext(ctx, "Failed to decode page_token, starting from beginning", "error", err)
		return nil
	}
	if id, ok := cursor.GetInt64("id"); ok {
		return &id
	}
	return nil
}
