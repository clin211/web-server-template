package user

import (
	"context"
	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/store/where"
	"testing"
)

func TestListOnlyReturnsCursorWhenAnotherPageExists(t *testing.T) {
	for _, rows := range []int{0, 2, 3} {
		t.Run(string(rune('0'+rows)), func(t *testing.T) {
			users := make([]*model.UserM, rows)
			for i := range users {
				users[i] = &model.UserM{ID: int64(10 - i), UserID: "member"}
			}
			biz := New(&fakeStore{user: &fakeUserStore{list: func(_ context.Context, opts *where.Options) (int64, []*model.UserM, error) {
				if opts.Limit != 3 {
					t.Errorf("query limit=%d, want one lookahead row", opts.Limit)
				}
				return int64(rows), users, nil
			}}}, nil)
			response, err := biz.List(context.Background(), &v1.ListUserRequest{PageSize: 2})
			if err != nil {
				t.Fatal(err)
			}
			if len(response.Users) != min(rows, 2) {
				t.Fatalf("returned %d rows", len(response.Users))
			}
			if (response.PageToken != "") != (rows > 2) {
				t.Fatalf("rows=%d token=%q", rows, response.PageToken)
			}
		})
	}
}
