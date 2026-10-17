package validation

import (
	"context"
	"errors"
	"testing"

	"github.com/clin211/gin-enterprise-template/internal/pkg/contextx"
	"github.com/clin211/gin-enterprise-template/internal/pkg/errno"
	v1 "github.com/clin211/gin-enterprise-template/pkg/api/apiserver/v1"
	"github.com/clin211/gin-enterprise-template/pkg/errorsx"
)

func TestUserResourceValidationEnforcesOwnership(t *testing.T) {
	validator := &Validator{}
	tests := []struct {
		name     string
		userID   string
		validate func(context.Context) error
		wantErr  error
	}{
		{
			name:   "get own user",
			userID: "user-1",
			validate: func(ctx context.Context) error {
				return validator.ValidateGetUserRequest(ctx, &v1.GetUserRequest{UserID: "user-1"})
			},
		},
		{
			name:   "get another user",
			userID: "user-1",
			validate: func(ctx context.Context) error {
				return validator.ValidateGetUserRequest(ctx, &v1.GetUserRequest{UserID: "user-2"})
			},
			wantErr: errno.ErrPermissionDenied,
		},
		{
			name:   "update another user",
			userID: "user-1",
			validate: func(ctx context.Context) error {
				return validator.ValidateUpdateUserRequest(ctx, &v1.UpdateUserRequest{UserID: "user-2"})
			},
			wantErr: errno.ErrPermissionDenied,
		},
		{
			name:   "delete another user",
			userID: "user-1",
			validate: func(ctx context.Context) error {
				return validator.ValidateDeleteUserRequest(ctx, &v1.DeleteUserRequest{UserID: "user-2"})
			},
			wantErr: errno.ErrPermissionDenied,
		},
		{
			name:   "change another users password",
			userID: "user-1",
			validate: func(ctx context.Context) error {
				return validator.ValidateChangePasswordRequest(ctx, &v1.ChangePasswordRequest{
					UserID:      "user-2",
					OldPassword: "old123",
					NewPassword: "new123",
				})
			},
			wantErr: errno.ErrPermissionDenied,
		},
		{
			name: "missing authenticated identity",
			validate: func(ctx context.Context) error {
				return validator.ValidateGetUserRequest(ctx, &v1.GetUserRequest{UserID: "user-1"})
			},
			wantErr: errno.ErrUnauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.userID != "" {
				ctx = contextx.WithUserID(ctx, tt.userID)
			}

			err := tt.validate(ctx)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("validate() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateListUserRequestDeniedWithoutAdministratorModel(t *testing.T) {
	validator := &Validator{}
	ctx := contextx.WithUserID(context.Background(), "user-1")

	err := validator.ValidateListUserRequest(ctx, &v1.ListUserRequest{})
	if !errors.Is(err, errno.ErrPermissionDenied) {
		t.Fatalf("ValidateListUserRequest() error = %v, want %v", err, errno.ErrPermissionDenied)
	}
}

func TestValidateUpdateUserRequestValidatesChangedFields(t *testing.T) {
	validator := &Validator{}
	ctx := contextx.WithUserID(context.Background(), "user-1")
	invalidEmail := "not-an-email"

	err := validator.ValidateUpdateUserRequest(ctx, &v1.UpdateUserRequest{
		UserID: "user-1",
		Email:  &invalidEmail,
	})
	if !errors.Is(err, errno.ErrInvalidArgument) {
		t.Fatalf("ValidateUpdateUserRequest() error = %v, want %v", err, errno.ErrInvalidArgument)
	}
}

func TestUserRulesRejectUnexpectedTypes(t *testing.T) {
	rules := (&Validator{}).ValidateUserRules()
	for _, name := range []string{"Password", "OldPassword", "NewPassword", "UserID", "Username", "Nickname", "Email", "Phone", "Limit"} {
		t.Run(name, func(t *testing.T) {
			if err := rules[name](struct{}{}); !errors.Is(err, errno.ErrInvalidArgument) {
				t.Fatalf("unexpected type error = %v, want invalid argument", err)
			}
		})
	}
}

func TestPasswordValidationPreservesExistingCredentials(t *testing.T) {
	validator := &Validator{}
	ctx := contextx.WithUserID(context.Background(), "user-1")
	for _, tt := range []struct {
		name     string
		validate func() error
		wantID   string
	}{
		{"legacy login", func() error {
			return validator.ValidateLoginRequest(ctx, &v1.LoginRequest{Username: "user1", Password: "a1😀"})
		}, ""},
		{"legacy old password", func() error {
			return validator.ValidateChangePasswordRequest(ctx, &v1.ChangePasswordRequest{UserID: "user-1", OldPassword: "a1😀", NewPassword: "new123"})
		}, ""},
		{"missing login password", func() error {
			return validator.ValidateLoginRequest(ctx, &v1.LoginRequest{Username: "user1"})
		}, "Validation.PasswordRequired"},
		{"missing old password", func() error {
			return validator.ValidateChangePasswordRequest(ctx, &v1.ChangePasswordRequest{UserID: "user-1", NewPassword: "new123"})
		}, "Validation.PasswordRequired"},
		{"short new password", func() error {
			return validator.ValidateChangePasswordRequest(ctx, &v1.ChangePasswordRequest{UserID: "user-1", OldPassword: "old123", NewPassword: "a1😀"})
		}, "Validation.PasswordTooShort"},
		{"valid Unicode new password", func() error {
			return validator.ValidateChangePasswordRequest(ctx, &v1.ChangePasswordRequest{UserID: "user-1", OldPassword: "a1😀", NewPassword: "a1中😀文好"})
		}, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate()
			if tt.wantID == "" {
				if err != nil {
					t.Fatalf("valid credential rejected: %v", err)
				}
				return
			}
			var bizErr *errorsx.BizError
			if !errors.As(err, &bizErr) || bizErr.MessageID != tt.wantID {
				t.Fatalf("validation error = %v, want message ID %s", err, tt.wantID)
			}
		})
	}
}
