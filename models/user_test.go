package models

import "testing"

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     RegisterRequest
		wantErr bool
	}{
		{
			name:    "empty name",
			req:     RegisterRequest{Name: "", Email: "ada@example.com", Password: "secret123", ConfirmPassword: "secret123"},
			wantErr: true,
		},
		{
			name:    "empty email",
			req:     RegisterRequest{Name: "Ada", Email: "", Password: "secret123", ConfirmPassword: "secret123"},
			wantErr: true,
		},
		{
			name:    "empty password",
			req:     RegisterRequest{Name: "Ada", Email: "ada@example.com", Password: "", ConfirmPassword: ""},
			wantErr: true,
		},
		{
			name:    "password too short",
			req:     RegisterRequest{Name: "Ada", Email: "ada@example.com", Password: "12345", ConfirmPassword: "12345"},
			wantErr: true,
		},
		{
			name:    "confirm does not match",
			req:     RegisterRequest{Name: "Ada", Email: "ada@example.com", Password: "secret123", ConfirmPassword: "secret124"},
			wantErr: true,
		},
		{
			name:    "valid request",
			req:     RegisterRequest{Name: "Ada", Email: "ada@example.com", Password: "secret123", ConfirmPassword: "secret123"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
