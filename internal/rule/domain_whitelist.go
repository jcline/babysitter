package rule

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"

	valid "github.com/asaskevich/govalidator"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
)

type DomainWhitelistConfig struct {
	Whitelist []string `json:whitelist`
}

func (dwc *DomainWhitelistConfig) String() string {
	var b strings.Builder
	for _, v := range dwc.Whitelist {
		fmt.Fprintf(&b, "%s, ", v)
	}
	return b.String()[:b.Len()-2]
}

func LoadWhitelist(path string) (*DomainWhitelistConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wbc DomainWhitelistConfig
	delimited := bytes.Split(contents, []byte("\n"))
	for _, entry := range delimited {
		// allow empty lines and comments
		if len(entry) == 0 || entry[0] == '#' {
			continue
		}
		strEntry := strings.TrimSpace(string(entry))
		if !valid.IsDNSName(strEntry) {
			return nil, fmt.Errorf(
				"invalid host in whitelist %v",
				entry)
		}
		wbc.Whitelist = append(wbc.Whitelist, strEntry)
	}

	sort.Strings(wbc.Whitelist)

	return &wbc, nil
}

func LoadWhitelistFromArray(array []string) (*DomainWhitelistConfig, error) {
	sort.Strings(array)
	return &DomainWhitelistConfig{
		Whitelist: array,
	}, nil
}

type DomainWhitelist struct {
	conf  *DomainWhitelistConfig
	re    *regexp.Regexp
	cache *lru.TwoQueueCache
}

func (dw *DomainWhitelist) String() string {
	return dw.conf.String()
}

func (dw *DomainWhitelist) allow(request *http.Request) (permitted, bool) {
	status, ok := dw.cache.Get(request.Host)
	if ok {
		result, ok := status.(permitted)
		if ok {
			return result, ok
		}
		// If !ok let's log the error and continue... effectively it
		// just means a disabled cache
		log.Error().Str("type", fmt.Sprintf("%T", result)).
			Msg("result was not of type permitted")
	}

	if dw.re.MatchString(request.Host) {
		dw.cache.Add(request.Host, allow)
		return allow, false
	}

	// If we don't have a rule for this domain we cannot presume it's
	// permissibility
	dw.cache.Add(request.Host, pass)
	return pass, false
}

//NewDomainWhitelist creates a DomainWhitelist or fails if the regular
//expression we build fails to compile.
func NewDomainWhitelist(
	config *DomainWhitelistConfig,
) (*DomainWhitelist, error) {
	dw := &DomainWhitelist{
		conf: config,
	}
	prefix := `(.*\.|)`
	pattern := "("
	for i := 0; i < len(config.Whitelist); i += 1 {
		domain := strings.Replace(
			config.Whitelist[i],
			".",
			"\\.",
			len(config.Whitelist[i]))
		if i == 0 {
			pattern = pattern + prefix + domain
		} else {
			pattern = pattern + "|" + prefix + domain
		}
	}
	pattern += ")"
	log.Debug().Str("pattern", pattern).Msg("whitelist matcher")

	var err error
	dw.re, err = regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	dw.cache, err = lru.New2Q(10000)
	if err != nil {
		return nil, err
	}

	return dw, nil
}
