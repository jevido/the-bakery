package limits

import "testing"

func TestClientIP(t *testing.T) {
	tests := []struct {
		name, remote, xff, want string
	}{
		{"direct client", "203.0.113.7:5000", "", "203.0.113.7"},
		{"a direct client cannot claim another address", "203.0.113.7:5000", "198.51.100.1", "203.0.113.7"},
		{"behind the proxy", "10.0.1.5:5000", "198.51.100.9", "198.51.100.9"},
		{"a spoofed hop is ignored", "10.0.1.5:5000", "1.2.3.4, 198.51.100.9", "198.51.100.9"},
		{"internal hops are skipped", "172.18.0.2:5000", "198.51.100.9, 10.0.0.3", "198.51.100.9"},
		{"local development", "127.0.0.1:5000", "", "127.0.0.1"},
		{"junk in the header", "10.0.1.5:5000", "nonsense", "10.0.1.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientIP(tt.remote, tt.xff); got != tt.want {
				t.Fatalf("clientIP(%q, %q) = %q, want %q", tt.remote, tt.xff, got, tt.want)
			}
		})
	}
}

func TestWait(t *testing.T) {
	for n, want := range map[int]string{0: "a second", 30: "30 seconds", 3599: "60 minutes", 7200: "2 hours", 86399: "24 hours"} {
		if got := wait(n); got != want {
			t.Errorf("wait(%d) = %q, want %q", n, got, want)
		}
	}
}
