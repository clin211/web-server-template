package user

import (
	"context"
	"log/slog"
	"sync"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/pkg/conversion"
	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/known"
	"github.com/clin211/gin-enterprise-template/internal/pkg/pagination"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"golang.org/x/sync/errgroup"

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

	var m sync.Map
	eg, ctx := errgroup.WithContext(ctx)

	// 设置最大并发数量为常量 MaxConcurrency
	eg.SetLimit(known.MaxErrGroupConcurrency)

	// 使用 goroutine 提高接口性能
	for _, user := range userList {
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return nil
			default:
				// 这里可以加入耗时的逻辑
				/*
					count, _, err := b.store.Posts().List(ctx, where.T(ctx))
					if err != nil {
						return err
					}
				*/

				userv1 := conversion.UserModelToUserV1(user)
				// userv1.PostCount = count
				m.Store(user.ID, userv1)

				return nil
			}
		})
	}

	if err := eg.Wait(); err != nil {
		slog.ErrorContext(ctx, "Failed to wait all function calls returned", "error", err)
		return nil, err
	}

	users := make([]*v1.User, 0, len(userList))
	for _, item := range userList {
		user, _ := m.Load(item.ID)
		users = append(users, user.(*v1.User))
	}

	// 生成下一页的 page_token
	// 只有当返回的数据量等于 pageSize 时，才说明可能有下一页
	var nextPageToken string
	if len(userList) == pageSize {
		lastUser := userList[len(userList)-1]
		cursor, err := pagination.NewCursor("id", lastUser.ID)
		if err == nil {
			nextPageToken, _ = cursor.Encode()
		}
	}

	slog.InfoContext(ctx, "Get users from backend storage", "count", len(users))

	return &v1.ListUserResponse{
		TotalCount: count,
		Users:      users,
		PageToken:  nextPageToken,
	}, nil
}
