package admin

import "testing"

func TestIntegrationAdminPathAllowed(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/accounts", want: true},
		{path: "/accounts/42/test", want: true},
		{path: "/groups/all", want: true},
		{path: "/proxies/all", want: true},
		{path: "/settings/web-search-emulation/test", want: true},
		{path: "/users", want: false},
		{path: "/payments", want: false},
		{path: "/system/update", want: false},
		{path: "/accounts-evil", want: false},
		{path: "/groups", want: false},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			if got := integrationAdminPathAllowed(test.path); got != test.want {
				t.Fatalf("integrationAdminPathAllowed(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}
