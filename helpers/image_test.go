package helpers

import "testing"

func TestDownloadImageAsBase64EmptyURL(t *testing.T) {
	got, err := DownloadImageAsBase64("")
	if err != nil {
		t.Fatalf("DownloadImageAsBase64(\"\") returned error: %v", err)
	}
	if got != "" {
		t.Errorf("DownloadImageAsBase64(\"\") = %q, want empty string", got)
	}
}

func TestDownloadImageAsBase64Rejects(t *testing.T) {
	// Every case below is rejected during parse/validate/resolve, before any
	// outbound HTTP request, so the test runs offline.
	tests := []struct {
		name string
		url  string
	}{
		{"http scheme", "http://example.com/x.png"},
		{"ftp scheme", "ftp://example.com/x.png"},
		{"no host", "https:///x.png"},
		{"loopback ip", "https://127.0.0.1/x.png"},
		{"metadata ip", "https://169.254.169.254/latest/meta-data"},
		{"private 10.x", "https://10.0.0.5/x.png"},
		{"private 192.168.x", "https://192.168.1.1/x.png"},
		{"unspecified", "https://0.0.0.0/x.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DownloadImageAsBase64(tt.url)
			if err == nil {
				t.Errorf("DownloadImageAsBase64(%q) = nil error, want error", tt.url)
			}
			if got != "" {
				t.Errorf("DownloadImageAsBase64(%q) = %q, want empty string", tt.url, got)
			}
		})
	}
}
