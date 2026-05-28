package user

import (
	"context"
	"log/slog"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
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
	if contextx.Username(ctx) != known.AdminUsername {
		whr.T(ctx)
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
