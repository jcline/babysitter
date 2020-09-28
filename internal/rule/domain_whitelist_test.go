package rule

import (
	"net/http/httptest"
	"testing"
)

func Test_Whitelist(t *testing.T) {
	config := DomainWhitelistConfig{
		Whitelist: []string{
			"example.com",
			"subdomain.example2.com",
		},
	}

	tests := map[string]permitted{
		"https://example.com":                          allow,
		"https://subdomain.example2.com":               allow,
		"https://cat.com":                              pass,
		"https://subdomain.example.com":                allow,
		"http://example.com":                           allow,
		"http://subdomain.example2.com":                allow,
		"http://cat.com":                               pass,
		"http://subdomain.example.com":                 allow,
		"https://api.twitter.com/1.1/branch/init.json": pass,
	}

	wl, err := NewDomainWhitelist(&config)
	if err != nil {
		t.Fatalf("got %v wanted nil", err)
	}

	for host, permit := range tests {
		result, _ := wl.allow(httptest.NewRequest("GET", host, nil))
		if result != permit {
			t.Fatalf("got %v, wanted %v for %v", result, permit, host)
		}
	}
}
