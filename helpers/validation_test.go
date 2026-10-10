package helpers

import "testing"

func TestIsValidCPF(t *testing.T) {
	tests := []struct {
		name string
		cpf  string
		want bool
	}{
		// Reference value, masked and unmasked.
		{"reference masked", "529.982.247-25", true},
		{"reference unmasked", "52998224725", true},
		// The 6 CPFs used in db/seed.go.
		{"seed 1", "529.982.247-25", true},
		{"seed 2", "168.995.350-09", true},
		{"seed 3", "111.444.777-35", true},
		{"seed 4", "390.533.447-05", true},
		{"seed 5", "457.845.818-05", true},
		{"seed 6", "764.831.158-57", true},
		// Invalid cases.
		{"wrong check digit", "529.982.247-24", false},
		{"too short", "5299822472", false},
		{"too long", "529982247250", false},
		{"empty", "", false},
		{"all repeated 1s", "11111111111", false},
		{"all repeated 0s", "00000000000", false},
		{"all repeated 9s", "99999999999", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidCPF(tt.cpf); got != tt.want {
				t.Errorf("IsValidCPF(%q) = %v, want %v", tt.cpf, got, tt.want)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"simple", "user@example.com", true},
		{"subdomain", "iann.barbosa@ccc.ufcg.edu.br", true},
		{"missing domain", "a@", false},
		{"no at sign", "sem-arroba", false},
		{"display name form", "Nome <a@b.com>", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidEmail(tt.email); got != tt.want {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, got, tt.want)
			}
		})
	}
}

func TestIsValidBRPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"10 digits unmasked", "1133334444", true},
		{"10 digits masked", "(11) 3333-4444", true},
		{"11 digits unmasked", "11987654321", true},
		{"11 digits masked", "(11) 98765-4321", true},
		{"9 digits", "113334444", false},
		{"12 digits", "119876543210", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidBRPhone(tt.phone); got != tt.want {
				t.Errorf("IsValidBRPhone(%q) = %v, want %v", tt.phone, got, tt.want)
			}
		})
	}
}

func TestIsValidCEP(t *testing.T) {
	tests := []struct {
		name string
		cep  string
		want bool
	}{
		{"8 digits unmasked", "01234567", true},
		{"8 digits masked", "01234-567", true},
		{"7 digits", "0123456", false},
		{"9 digits", "012345678", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidCEP(tt.cep); got != tt.want {
				t.Errorf("IsValidCEP(%q) = %v, want %v", tt.cep, got, tt.want)
			}
		})
	}
}
