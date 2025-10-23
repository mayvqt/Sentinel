package validation

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with plus", "test+label@example.com", false},
		{"valid email with subdomain", "test@mail.example.com", false},
		{"empty email", "", true},
		{"missing @", "testexample.com", true},
		{"missing domain", "test@", true},
		{"missing local part", "@example.com", true},
		{"invalid characters", "test@ex ample.com", true},
		{"too long", string(make([]byte, 256)) + "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid username", "testuser", false},
		{"valid with numbers", "user123", false},
		{"valid with underscore", "test_user", false},
		{"valid with hyphen", "test-user", false},
		{"empty username", "", true},
		{"too short", "ab", true},
		{"too long", string(make([]byte, 33)), true},
		{"invalid characters", "test@user", true},
		{"reserved username", "admin", true},
		{"reserved username case insensitive", "ADMIN", true},
		{"starts with number", "123user", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid strong password", "Password123!", false},
		{"valid with special chars", "MySecure@Pass1", false},
		{"empty password", "", true},
		{"too short", "Pass1!", true},
		{"too long", string(make([]byte, 129)), true},
		{"no uppercase", "password123!", true},
		{"no lowercase", "PASSWORD123!", true},
		{"no numbers", "Password!", true},
		{"no special chars", "Password123", true},
		{"common password", "password", true},
		{"common password case", "Password123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRole(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
	}{
		{"valid user role", "user", false},
		{"valid admin role", "admin", false},
		{"valid moderator role", "moderator", false},
		{"empty role", "", true},
		{"invalid role", "superuser", true},
		{"case sensitive", "User", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRole(tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRole() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal input", "hello world", "hello world"},
		{"with whitespace", "  hello world  ", "hello world"},
		{"with null bytes", "hello\x00world", "helloworld"},
		{"with control chars", "hello\x01\x02world", "helloworld"},
		{"with tabs and newlines", "hello\tworld\n", "hello\tworld\n"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name     string
		username string
		email    string
		password string
		wantErr  bool
	}{
		{
			"valid request",
			"testuser",
			"test@example.com",
			"SecurePass123!",
			false,
		},
		{
			"invalid username",
			"ab",
			"test@example.com",
			"SecurePass123!",
			true,
		},
		{
			"invalid email",
			"testuser",
			"invalid-email",
			"SecurePass123!",
			true,
		},
		{
			"weak password",
			"testuser",
			"test@example.com",
			"weak",
			true,
		},
		{
			"multiple validation errors",
			"",
			"invalid-email",
			"weak",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegisterRequest(tt.username, tt.email, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func BenchmarkValidatePassword(b *testing.B) {
	password := "SecurePassword123!"
	for i := 0; i < b.N; i++ {
		ValidatePassword(password)
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := "test@example.com"
	for i := 0; i < b.N; i++ {
		ValidateEmail(email)
	}
}
