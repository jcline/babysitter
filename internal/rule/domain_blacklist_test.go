package rule

import (
	"net/http/httptest"
	"testing"
)

func Test_Blacklist(t *testing.T) {
	config := DomainBlacklistConfig{
		Blacklist: []string{
			"example.com",
			"subdomain.example2.com",
			"twitter.com",
		},
	}

	tests := map[string]permitted{
		"https://example.com":                          deny,
		"https://subdomain.example2.com":               deny,
		"https://cat.com":                              pass,
		"https://subdomain.example.com":                deny,
		"http://example.com":                           deny,
		"http://subdomain.example2.com":                deny,
		"http://cat.com":                               pass,
		"http://subdomain.example.com":                 deny,
		"https://api.twitter.com/1.1/branch/init.json": deny,
	}

	bl, err := NewDomainBlacklist(&config)
	if err != nil {
		t.Fatalf("got %v wanted nil", err)
	}

	for host, permit := range tests {
		result, _ := bl.allow(httptest.NewRequest("GET", host, nil))
		if result != permit {
			t.Fatalf("got %v, wanted %v for %v", result, permit, host)
		}
	}
}
