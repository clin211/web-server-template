package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

func TestUserModelToUserV1(t *testing.T) {
	email := "alice@example.com"
	phone := "13800138000"
	avatar := "https://example.com/avatar.png"
	description := "hello"
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)
	lastLoginAt := time.Unix(1700007200, 0)

	tests := []struct {
		name            string
		user            *model.UserM
		wantUser        string
		wantName        string
		wantNick        string
		wantEmail       string
		wantPhone       string
		wantAvatar      string
		wantDescription string
		wantStatus      int32
		wantGender      int32
		wantLastLoginAt int64
		wantCAt         int64
		wantUAt         int64
	}{
		{
			name:            "maps populated fields",
			user:            &model.UserM{UserID: "user-1", Username: "alice", Nickname: "Alice", Email: &email, Phone: &phone, Avatar: &avatar, Description: &description, Status: 1, Gender: 2, LastLoginAt: &lastLoginAt, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantUser:        "user-1",
			wantName:        "alice",
			wantNick:        "Alice",
			wantEmail:       email,
			wantPhone:       phone,
			wantAvatar:      avatar,
			wantDescription: description,
			wantStatus:      1,
			wantGender:      2,
			wantLastLoginAt: lastLoginAt.Unix(),
			wantCAt:         createdAt.Unix(),
			wantUAt:         updatedAt.Unix(),
		},
		{
			name:            "maps nil pointers to empty strings and zero time",
			user:            &model.UserM{UserID: "user-2", Username: "bob", Nickname: "Bob", CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantUser:        "user-2",
			wantName:        "bob",
			wantNick:        "Bob",
			wantEmail:       "",
			wantPhone:       "",
			wantAvatar:      "",
			wantDescription: "",
			wantLastLoginAt: 0,
			wantCAt:         createdAt.Unix(),
			wantUAt:         updatedAt.Unix(),
		},
		{
			name:            "returns empty user for nil input",
			user:            nil,
			wantEmail:       "",
			wantPhone:       "",
			wantAvatar:      "",
			wantDescription: "",
			wantLastLoginAt: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UserModelToUserV1(tt.user)
			if got == nil {
				t.Fatal("UserModelToUserV1() returned nil")
			}
			if got.UserID != tt.wantUser {
				t.Fatalf("UserID = %q, want %q", got.UserID, tt.wantUser)
			}
			if got.Username != tt.wantName {
				t.Fatalf("Username = %q, want %q", got.Username, tt.wantName)
			}
			if got.Nickname != tt.wantNick {
				t.Fatalf("Nickname = %q, want %q", got.Nickname, tt.wantNick)
			}
			if got.Email != tt.wantEmail {
				t.Fatalf("Email = %q, want %q", got.Email, tt.wantEmail)
			}
			if got.Phone != tt.wantPhone {
				t.Fatalf("Phone = %q, want %q", got.Phone, tt.wantPhone)
			}
			if got.Avatar != tt.wantAvatar {
				t.Fatalf("Avatar = %q, want %q", got.Avatar, tt.wantAvatar)
			}
			if got.Description != tt.wantDescription {
				t.Fatalf("Description = %q, want %q", got.Description, tt.wantDescription)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("Status = %d, want %d", got.Status, tt.wantStatus)
			}
			if got.Gender != tt.wantGender {
				t.Fatalf("Gender = %d, want %d", got.Gender, tt.wantGender)
			}
			if got.LastLoginAt != tt.wantLastLoginAt {
				t.Fatalf("LastLoginAt = %d, want %d", got.LastLoginAt, tt.wantLastLoginAt)
			}
			if got.CreatedAt != tt.wantCAt {
				t.Fatalf("CreatedAt = %d, want %d", got.CreatedAt, tt.wantCAt)
			}
			if got.UpdatedAt != tt.wantUAt {
				t.Fatalf("UpdatedAt = %d, want %d", got.UpdatedAt, tt.wantUAt)
			}
			if got.PostCount != 0 {
				t.Fatalf("PostCount = %d, want 0", got.PostCount)
			}
		})
	}
}
