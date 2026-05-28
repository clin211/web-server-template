package authn

import (
	"testing"
)

// 定义测试用例结构
type encryptTestCase struct {
	name    string
	source  string
	wantErr bool
}

// TestEncrypt 测试密码加密功能
func TestEncrypt(t *testing.T) {
	tests := []encryptTestCase{
		{
			name:    "正常密码加密",
			source:  "password123",
			wantErr: false,
		},
		{
			name:    "空密码加密",
			source:  "",
			wantErr: false,
		},
		{
			name:    "复杂密码加密",
			source:  "P@ssw0rd!#$%",
			wantErr: false,
		},
		{
			name:    "中文密码加密",
			source:  "密码123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashed, err := Encrypt(tt.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 验证哈希后的密码不为空
			if hashed == "" {
				t.Error("Encrypt() 返回的哈希值为空")
			}

			// 验证哈希后的密码与原密码不同
			if hashed == tt.source {
				t.Error("Encrypt() 哈希后的密码与原密码相同")
			}
		})
	}
}

// compareTestCase 定义比较密码的测试用例
type compareTestCase struct {
	name           string
	hashedPassword string
	password       string
	wantErr        bool
}

// TestCompare 测试密码比较功能
func TestCompare(t *testing.T) {
	// 先生成一些哈希密码用于测试
	hash1, _ := Encrypt("password123")
	hash2, _ := Encrypt("P@ssw0rd!#$%")
	hash3, _ := Encrypt("密码123")

	tests := []compareTestCase{
		{
			name:           "密码匹配",
			hashedPassword: hash1,
			password:       "password123",
			wantErr:        false,
		},
		{
			name:           "密码不匹配",
			hashedPassword: hash1,
			password:       "wrongpassword",
			wantErr:        true,
		},
		{
			name:           "复杂密码匹配",
			hashedPassword: hash2,
			password:       "P@ssw0rd!#$%",
			wantErr:        false,
		},
		{
			name:           "中文密码匹配",
			hashedPassword: hash3,
			password:       "密码123",
			wantErr:        false,
		},
		{
			name:           "空密码不匹配",
			hashedPassword: hash1,
			password:       "",
			wantErr:        true,
		},
		{
			name:           "无效哈希格式",
			hashedPassword: "invalidhash",
			password:       "password123",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Compare(tt.hashedPassword, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestEncryptAndConsistency 测试加密的一致性
func TestEncryptAndConsistency(t *testing.T) {
	source := "testpassword"

	// 同一个密码加密两次应该产生不同的哈希值（因为 bcrypt 包含随机盐）
	hash1, err1 := Encrypt(source)
	hash2, err2 := Encrypt(source)

	if err1 != nil || err2 != nil {
		t.Fatalf("Encrypt() 失败: err1=%v, err2=%v", err1, err2)
	}

	// 哈希值应该不同
	if hash1 == hash2 {
		t.Error("同一密码两次加密应产生不同的哈希值（盐值不同）")
	}

	// 但两个哈希值都应该能匹配原密码
	if err := Compare(hash1, source); err != nil {
		t.Errorf("hash1 无法匹配原密码: %v", err)
	}
	if err := Compare(hash2, source); err != nil {
		t.Errorf("hash2 无法匹配原密码: %v", err)
	}
}
