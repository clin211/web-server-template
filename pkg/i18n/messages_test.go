package i18n

import "testing"

func TestDefaultMessage(t *testing.T) {
	for _, tt := range []struct{ id, want string }{
		{"Common.InternalError", "内部服务器错误。"},
		{"User.NotFound", "用户未找到。"},
		{"Auth.TokenExpired", "令牌已过期。"},
		{"missing.message", "missing.message"},
	} {
		t.Run(tt.id, func(t *testing.T) {
			if got := DefaultMessage(tt.id); got != tt.want {
				t.Fatalf("DefaultMessage(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
