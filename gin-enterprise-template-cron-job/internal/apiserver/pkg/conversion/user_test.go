package conversion

import (
	"testing"
	"time"

	"github.com/clin211/gin-enterprise-template/internal/apiserver/model"
)

func TestUserModelToUserV1(t *testing.T) {
	email := "alice@example.com"
	phone := "13800138000"
	createdAt := time.Unix(1700000000, 0)
	updatedAt := time.Unix(1700003600, 0)

	tests := []struct {
		name      string
		user      *model.UserM
		wantUser  string
		wantName  string
		wantNick  string
		wantEmail string
		wantPhone string
		wantCAt   int64
		wantUAt   int64
	}{
		{
			name:      "maps populated fields",
			user:      &model.UserM{UserID: "user-1", Username: "alice", Nickname: "Alice", Email: &email, Phone: &phone, CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantUser:  "user-1",
			wantName:  "alice",
			wantNick:  "Alice",
			wantEmail: email,
			wantPhone: phone,
			wantCAt:   createdAt.Unix(),
			wantUAt:   updatedAt.Unix(),
		},
		{
			name:      "maps nil pointers to empty strings",
			user:      &model.UserM{UserID: "user-2", Username: "bob", Nickname: "Bob", CreatedAt: createdAt, UpdatedAt: updatedAt},
			wantUser:  "user-2",
			wantName:  "bob",
			wantNick:  "Bob",
			wantEmail: "",
			wantPhone: "",
			wantCAt:   createdAt.Unix(),
			wantUAt:   updatedAt.Unix(),
		},
		{
			name:      "returns empty user for nil input",
			user:      nil,
			wantEmail: "",
			wantPhone: "",
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
