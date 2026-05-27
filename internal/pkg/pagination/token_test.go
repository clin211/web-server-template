package pagination

import "testing"

func TestNextPageToken(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		pageSize   int
		lastID     int64
		wantEmpty  bool
		wantID     int64
		wantCalled bool
	}{
		{
			name:       "empty result does not generate token",
			length:     0,
			pageSize:   10,
			lastID:     101,
			wantEmpty:  true,
			wantCalled: false,
		},
		{
			name:       "short page does not generate token",
			length:     9,
			pageSize:   10,
			lastID:     102,
			wantEmpty:  true,
			wantCalled: false,
		},
		{
			name:       "full page generates token",
			length:     10,
			pageSize:   10,
			lastID:     103,
			wantEmpty:  false,
			wantID:     103,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			got := NextPageToken(tt.length, tt.pageSize, func() int64 {
				called = true
				return tt.lastID
			})

			if called != tt.wantCalled {
				t.Fatalf("called = %v, want %v", called, tt.wantCalled)
			}

			if tt.wantEmpty {
				if got != "" {
					t.Fatalf("NextPageToken() = %q, want empty", got)
				}
				return
			}

			if got == "" {
				t.Fatal("NextPageToken() returned empty token")
			}

			cursor, err := DecodeCursor(got)
			if err != nil {
				t.Fatalf("DecodeCursor() error = %v", err)
			}

			gotID, ok := cursor.GetInt64("id")
			if !ok {
				t.Fatal("cursor.GetInt64(id) returned false")
			}
			if gotID != tt.wantID {
				t.Fatalf("cursor id = %d, want %d", gotID, tt.wantID)
			}
		})
	}
}
