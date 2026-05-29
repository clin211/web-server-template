package conversion

import (
	"time"

	"github.com/clin211/gin-enterprise-template/pkg/core"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
)

// UserModelToUserV1 将模型层的 UserM（用户模型对象）转换为 Protobuf 层的 User（v1 用户对象）.
func UserModelToUserV1(userModel *model.UserM) *v1.User {
	if userModel == nil {
		return &v1.User{}
	}

	return &v1.User{
		UserID:      userModel.UserID,
		Username:    userModel.Username,
		Nickname:    userModel.Nickname,
		Email:       derefString(userModel.Email),
		Phone:       derefString(userModel.Phone),
		CreatedAt:   userModel.CreatedAt.Unix(),
		UpdatedAt:   userModel.UpdatedAt.Unix(),
		Status:      int32(userModel.Status),
		Gender:      int32(userModel.Gender),
		Avatar:      derefString(userModel.Avatar),
		Description: derefString(userModel.Description),
		LastLoginAt: derefTime(userModel.LastLoginAt),
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefTime(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.Unix()
}

// UserV1ToUserModel 将 Protobuf 层的 User（v1 用户对象）转换为模型层的 UserM（用户模型对象）.
func UserV1ToUserModel(protoUser *v1.User) *model.UserM {
	var userModel model.UserM
	_ = core.CopyWithConverters(&userModel, protoUser)
	return &userModel
}

// UserModelListToUserV1List 将用户模型列表转换为 Protobuf 列表.
func UserModelListToUserV1List(users []*model.UserM) []*v1.User {
	result := make([]*v1.User, len(users))
	for i, user := range users {
		result[i] = UserModelToUserV1(user)
	}
	return result
}
