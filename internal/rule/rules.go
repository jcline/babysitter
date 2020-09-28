package rule

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type permitted int

const (
	allow permitted = iota + 1 // permit the request
	deny                       // deny the request
	pass                       // rule cannot decide
)

func (p permitted) String() string {
	switch p {
	case allow:
		return "allow"
	case deny:
		return "deny"
	case pass:
		return "pass"
	}
	return "invalid p"
}

type RuleConfig struct {
	*DomainBlacklistConfig
	*DomainWhitelistConfig
}

func (rc *RuleConfig) String() string {
	var b strings.Builder
	if rc.DomainBlacklistConfig != nil {
		b.WriteString("blacklist: ")
		b.WriteString(rc.DomainBlacklistConfig.String())
		b.WriteString("\n")
	}
	if rc.DomainWhitelistConfig != nil {
		b.WriteString("whitelist: ")
		b.WriteString(rc.DomainWhitelistConfig.String())
		b.WriteString("\n")
	}
	return b.String()
}

func NewRuleConfig(wlp, blp string) (*RuleConfig, error) {
	bl, err := LoadBlacklist(blp)
	if err != nil {
		return nil, fmt.Errorf("could not load blacklist: %v", err)
	}

	wl, err := LoadWhitelist(wlp)
	if err != nil {
		return nil, fmt.Errorf("could not load blacklist: %v", err)
	}

	rc := RuleConfig{
		DomainBlacklistConfig: bl,
		DomainWhitelistConfig: wl,
	}

	return &rc, nil
}

func NewRuleConfigFromMap(m map[string][]string) (*RuleConfig, error) {
	var err error

	rc := RuleConfig{}

	for k, v := range m {
		switch k {
		case "blacklist":
			rc.DomainBlacklistConfig, err = LoadBlacklistFromArray(v)
		case "whitelist":
			rc.DomainWhitelistConfig, err = LoadWhitelistFromArray(v)
		}

		if err != nil {
			return nil, fmt.Errorf("could not load %s: %v", k, err)
		}
	}

	return &rc, nil
}

type rule interface {
	allow(request *http.Request) (permitted, bool)
	fmt.Stringer
}

type Manager struct {
	rules map[string]rule
	conf  *RuleConfig
	lock  *sync.RWMutex
}

func NewManager() (*Manager, error) {
	return &Manager{
		rules: make(map[string]rule),
		lock:  &sync.RWMutex{},
	}, nil
}

func (rm *Manager) GetRules() *RuleConfig {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	result := *rm.conf

	return &result
}

// updateConfInLock updates Manager.conf, it assumes that it is only called
// inside the write lock
func (rm *Manager) updateConfInLock(rc *RuleConfig) error {
	if rm.conf == nil {
		rm.conf = rc
	} else {
		if rc.DomainBlacklistConfig != nil {
			rm.conf.DomainBlacklistConfig = rc.DomainBlacklistConfig
		}
		if rc.DomainWhitelistConfig != nil {
			rm.conf.DomainWhitelistConfig = rc.DomainWhitelistConfig
		}
	}

	return nil
}

//update the rule map with new rules in a threadsafe manner
func (rm *Manager) update(rules map[string]rule, rc *RuleConfig) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	for k, v := range rules {
		rm.rules[k] = v
	}

	rm.updateConfInLock(rc)
	return nil
}

func (rm *Manager) Update(rc *RuleConfig) error {
	var err error

	log.Info().Msg("updating config")
	defer log.Info().Msg("updated config")

	newRules := make(map[string]rule)

	if rc.DomainBlacklistConfig != nil {
		bl, err := NewDomainBlacklist(rc.DomainBlacklistConfig)
		if err != nil {
			return err
		}
		newRules["blacklist"] = bl
	}

	if rc.DomainWhitelistConfig != nil {
		wl, err := NewDomainWhitelist(rc.DomainWhitelistConfig)
		if err != nil {
			return err
		}
		newRules["whitelist"] = wl
	}

	err = rm.update(newRules, rc)
	if err != nil {
		return err
	}

	return nil
}

func (rm *Manager) Allow(request *http.Request) bool {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	// default allow
	ret := true
	for name, r := range rm.rules {
		status, cached := r.allow(request)
		if e := log.Debug(); e.Enabled() {
			e.Str("rule", name).
				Str("uri", request.URL.String()).
				Str("status", status.String()).
				Bool("cached", cached).
				Msg("applied rule")
		}

		switch status {
		case allow:
			// If we've whitelisted it all's good
			return true
		case deny:
			// If we ever fail a check then we'll mark it as failed
			ret = false
		case pass:
			// this rule didn't apply
			continue
		}
	}

	return ret
}

var RuleManager *Manager

func init() {
	var err error
	RuleManager, err = NewManager()
	if err != nil {
		panic(fmt.Sprintf("Could not create maanger %v", err))
	}
}
