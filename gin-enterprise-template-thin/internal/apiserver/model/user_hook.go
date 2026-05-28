package model

import (
	"github.com/clin211/gin-enterprise-template/pkg/authn"
	"gorm.io/gorm"
)

// BeforeCreate 在创建数据库记录之前加密明文密码。
func (m *UserM) BeforeCreate(tx *gorm.DB) error {
	// 加密用户密码。
	var err error
	m.Password, err = authn.Encrypt(m.Password)
	if err != nil {
		return err
	}

	return nil
}
