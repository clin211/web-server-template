package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

var userConversionResult *v1.User
var userListConversionResult []*v1.User

func BenchmarkUserModelToUserV1(b *testing.B) {
	email := "alice@example.com"
	phone := "13800138000"
	avatar := "https://example.com/avatar.png"
	description := "hello"
	lastLoginAt := time.Unix(1700007200, 0)
	user := &model.UserM{
		ID:          1,
		UserID:      "user-1",
		Username:    "alice",
		Nickname:    "Alice",
		Email:       &email,
		Phone:       &phone,
		Avatar:      &avatar,
		Description: &description,
		Gender:      1,
		Status:      0,
		LastLoginAt: &lastLoginAt,
		CreatedAt:   time.Unix(1700000000, 0),
		UpdatedAt:   time.Unix(1700003600, 0),
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		userConversionResult = UserModelToUserV1(user)
	}
}

func BenchmarkUserModelListToUserV1List(b *testing.B) {
	email := "alice@example.com"
	phone := "13800138000"
	users := make([]*model.UserM, 100)
	for i := range users {
		users[i] = &model.UserM{
			ID:        int64(i + 1),
			UserID:    "user",
			Username:  "username",
			Nickname:  "nickname",
			Email:     &email,
			Phone:     &phone,
			Gender:    1,
			Status:    0,
			CreatedAt: time.Unix(1700000000+int64(i), 0),
			UpdatedAt: time.Unix(1700003600+int64(i), 0),
		}
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		userListConversionResult = UserModelListToUserV1List(users)
	}
}
