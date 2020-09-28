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

type DomainBlacklistConfig struct {
	Blacklist []string `json:blacklist`
}

func (dbc *DomainBlacklistConfig) String() string {
	var b strings.Builder
	for _, v := range dbc.Blacklist {
		fmt.Fprintf(&b, "%s, ", v)
	}
	return b.String()[:b.Len()-2]
}

func LoadBlacklist(path string) (*DomainBlacklistConfig, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dbc DomainBlacklistConfig
	delimited := bytes.Split(contents, []byte("\n"))
	for _, entry := range delimited {
		// allow empty lines and comments
		if len(entry) == 0 || entry[0] == '#' {
			continue
		}
		strEntry := strings.TrimSpace(string(entry))
		if !valid.IsDNSName(strEntry) {
			return nil, fmt.Errorf(
				"invalid host in blacklist %v",
				entry)
		}
		dbc.Blacklist = append(dbc.Blacklist, strEntry)
	}

	sort.Strings(dbc.Blacklist)

	return &dbc, nil
}

func LoadBlacklistFromArray(array []string) (*DomainBlacklistConfig, error) {
	sort.Strings(array)
	return &DomainBlacklistConfig{
		Blacklist: array,
	}, nil
}

type DomainBlacklist struct {
	conf  *DomainBlacklistConfig
	re    *regexp.Regexp
	cache *lru.TwoQueueCache
}

func (db *DomainBlacklist) String() string {
	return db.conf.String()
}

func (db *DomainBlacklist) allow(request *http.Request) (permitted, bool) {
	status, ok := db.cache.Get(request.Host)
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

	if db.re.MatchString(request.Host) {
		db.cache.Add(request.Host, deny)
		return deny, false
	}

	// If we don't have a rule for this domain we cannot presume it's
	// permissibility
	db.cache.Add(request.Host, pass)
	return pass, false
}

//NewDomainBlacklist creates a DomainBlacklist or fails if the regular
//expression we build fails to compile.
func NewDomainBlacklist(
	config *DomainBlacklistConfig,
) (*DomainBlacklist, error) {
	db := &DomainBlacklist{
		conf: config,
	}
	prefix := `(.*\.|)`
	pattern := "("
	for i := 0; i < len(config.Blacklist); i += 1 {
		domain := strings.Replace(
			config.Blacklist[i],
			".",
			"\\.",
			len(config.Blacklist[i]))
		if i == 0 {
			pattern = pattern + prefix + domain
		} else {
			pattern = pattern + "|" + prefix + domain
		}
	}
	pattern += ")"
	log.Debug().Str("pattern", pattern).Msg("blacklist matcher")

	var err error
	db.re, err = regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	db.cache, err = lru.New2Q(10000)
	if err != nil {
		return nil, err
	}

	return db, nil
}
