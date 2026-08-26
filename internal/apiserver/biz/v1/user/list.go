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
	pageToken := rq.GetPageToken()
	var cursor *int64
	if pageToken != "" {
		decodedCursor, err := pagination.DecodeCursor(pageToken)
		if err != nil {
			// 当前 API 对非法/过期 token 采用兼容策略，回到第一页而不是拒绝请求；调用方
			// 可能看到重复结果，因此不能把 page_token 当作 exactly-once 遍历保证。
			slog.WarnContext(ctx, "Failed to decode page_token, starting from beginning", "error", err)
		} else {
			if id, ok := decodedCursor.GetInt64("id"); ok {
				cursor = &id
			}
		}
	}

	// 构建 where.Options，使用游标分页
	pageSize := pagination.NormalizePageSize(rq.GetPageSize())

	whr := where.NewWhere(where.WithLimit(int64(pageSize)))
	if cursor != nil {
		whr.Cursor = cursor
	}

	count, userList, err := b.store.User().List(ctx, whr)
	if err != nil {
		return nil, err
	}

	users := conversion.UserModelListToUserV1List(userList)

	nextPageToken := pagination.NextPageToken(len(userList), pageSize, func() int64 {
		return userList[len(userList)-1].ID
	})

	slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

	return &v1.ListUserResponse{
		TotalCount: count,
		Users:      users,
		PageToken:  nextPageToken,
	}, nil
}
