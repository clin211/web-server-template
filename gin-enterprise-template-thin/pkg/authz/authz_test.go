package authz

import (
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 创建测试用的 MySQL 数据库。
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := "root:@tcp(127.0.0.1:3306)/gin_enterprise_template?charset=utf8&parseTime=true&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("无法连接到测试数据库: %v (请确保 MySQL 服务正在运行)", err)
	}
	return db
}

// TestNewAuthz 测试创建授权器。
func TestNewAuthz(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "使用默认配置创建授权器",
			opts:    nil,
			wantErr: false,
		},
		{
			name: "使用自定义选项创建授权器",
			opts: []Option{
				WithAclModel(defaultAclModel),
				WithAutoLoadPolicyTime(5 * time.Second),
			},
			wantErr: false,
		},
		{
			name: "使用较短自动加载间隔",
			opts: []Option{
				WithAutoLoadPolicyTime(1 * time.Second),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			authz, err := NewAuthz(db, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAuthz() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && authz == nil {
				t.Error("NewAuthz() 返回 nil，但预期不为 nil")
			}
			if authz != nil {
				authz.StopAutoLoadPolicy()
			}
		})
	}
}

// TestAuthz_Authorize 测试授权功能。
func TestAuthz_Authorize(t *testing.T) {
	db := setupTestDB(t)
	authz, err := NewAuthz(db)
	if err != nil {
		t.Fatalf("无法创建授权器: %v", err)
	}
	defer authz.StopAutoLoadPolicy()

	tests := []struct {
		name    string
		sub     string
		obj     string
		act     string
		want    bool
		setup   func() // setup 函数用于在测试前设置策略
		cleanup func() // cleanup 函数用于在测试后清理策略
	}{
		{
			name:    "未设置策略时默认允许（allow-overrides 模型）",
			sub:     "alice",
			obj:     "data1",
			act:     "read",
			want:    true,
			setup:   func() {},
			cleanup: func() {},
		},
		{
			name: "允许读取权限",
			sub:  "alice",
			obj:  "data1",
			act:  "read",
			want: true,
			setup: func() {
				// Casbin 模型策略定义: p = sub, obj, act, eft
				authz.AddPolicy("alice", "data1", "read", "allow")
			},
			cleanup: func() {
				authz.RemovePolicy("alice", "data1", "read", "allow")
			},
		},
		{
			name: "显式拒绝未授权的操作",
			sub:  "alice",
			obj:  "data1",
			act:  "write",
			want: false,
			setup: func() {
				// 允许 read
				authz.AddPolicy("alice", "data1", "read", "allow")
				// 显式拒绝 write
				authz.AddPolicy("alice", "data1", "write", "deny")
			},
			cleanup: func() {
				authz.RemovePolicy("alice", "data1", "read", "allow")
				authz.RemovePolicy("alice", "data1", "write", "deny")
			},
		},
		{
			name: "允许通配符资源访问",
			sub:  "bob",
			obj:  "/datasets/123",
			act:  "read",
			want: true,
			setup: func() {
				// 添加通配符策略
				authz.AddPolicy("bob", "/datasets/*", "read", "allow")
			},
			cleanup: func() {
				authz.RemovePolicy("bob", "/datasets/*", "read", "allow")
			},
		},
		{
			name: "显式拒绝策略",
			sub:  "charlie",
			obj:  "admin",
			act:  "write",
			want: false,
			setup: func() {
				authz.AddPolicy("charlie", "admin", "write", "deny")
			},
			cleanup: func() {
				authz.RemovePolicy("charlie", "admin", "write", "deny")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got, err := authz.Authorize(tt.sub, tt.obj, tt.act)
			if err != nil {
				t.Errorf("Authorize() 意外错误: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("Authorize() = %v, want %v", got, tt.want)
			}
			tt.cleanup()
		})
	}
}

// TestWithAclModel 测试自定义 ACL 模型选项。
func TestWithAclModel(t *testing.T) {
	customModel := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act]`

	cfg := &authzConfig{}
	WithAclModel(customModel)(cfg)

	if cfg.aclModel != customModel {
		t.Errorf("WithAclModel() 模型未正确设置，got = %v, want = %v", cfg.aclModel, customModel)
	}
}

// TestWithAutoLoadPolicyTime 测试自动加载策略时间选项。
func TestWithAutoLoadPolicyTime(t *testing.T) {
	interval := 15 * time.Second
	cfg := &authzConfig{}
	WithAutoLoadPolicyTime(interval)(cfg)

	if cfg.autoLoadPolicyTime != interval {
		t.Errorf("WithAutoLoadPolicyTime() 时间未正确设置，got = %v, want = %v", cfg.autoLoadPolicyTime, interval)
	}
}

// TestDefaultOptions 测试默认选项。
func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts == nil {
		t.Error("DefaultOptions() 返回 nil")
		return
	}

	// 验证选项数量（默认包含 ACL 模型和自动加载时间）
	if len(opts) < 2 {
		t.Errorf("DefaultOptions() 选项数量不足，got = %d, want >= 2", len(opts))
	}
}

// TestProviderSet 测试 Wire ProviderSet。
func TestProviderSet(t *testing.T) {
	// Wire 的 ProviderSet 应该被正确定义
	// 验证 ProviderSet 不为零值
	var emptySet interface{} = ProviderSet
	if emptySet == nil {
		t.Error("ProviderSet 应该被定义")
	}
}

// benchmarkAuthorize 基准测试授权性能。
func benchmarkAuthorize(b *testing.B, sub, obj, act string) {
	db := setupTestDB(&testing.T{})
	authz, _ := NewAuthz(db)
	defer authz.StopAutoLoadPolicy()
	authz.AddPolicy("alice", "data1", "read", "allow")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		authz.Authorize(sub, obj, act)
	}
}

func BenchmarkAuthz_Authorize_Allow(b *testing.B) {
	benchmarkAuthorize(b, "alice", "data1", "read")
}

func BenchmarkAuthz_Authorize_Deny(b *testing.B) {
	benchmarkAuthorize(b, "bob", "data1", "read")
}
