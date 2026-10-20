package task

import (
	"context"
	"errors"

	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	"testing"
)

func TestRBACReadsRequireAuthentication(t *testing.T) {
	service := New(nil)
	for _, tc := range []struct {
		name string
		read func() error
	}{
		{"role", func() error { _, err := service.Role(context.Background(), "role_admin"); return err }},
		{"user roles", func() error { _, err := service.UserRoles(context.Background(), "owner"); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.read(); !errors.Is(err, errno.ErrUnauthenticated) {
				t.Fatalf("unauthenticated read = %v", err)
			}
		})
	}
}
