package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	valid "github.com/asaskevich/govalidator"

	"github.com/jcline/babysitter/internal/rule"
)

type strArray []string

func (sa *strArray) String() string {
	return strings.Join([]string(*sa), ",")
}

func (sa *strArray) Set(value string) error {
	values := strings.Split(value, ",")
	for _, v := range values {
		fmt.Printf("%s\n", v)
		if !valid.IsDNSName(v) {
			return fmt.Errorf("'%s' is not a valid hostname", v)
		}
	}
	*sa = append(*sa, values...)
	return nil
}

func (sa *strArray) Len() int {
	return len([]string(*sa))
}

func (sa *strArray) Merge(other []string) {
	*sa = append(*sa, other...)
}

type domain string

func (d *domain) String() string {
	return string(*d)
}

func (d *domain) Set(value string) error {
	if !valid.IsURL(value) {
		return fmt.Errorf("'%s' is not a valid url", value)
	}
	*d = domain(value)
	return nil
}

type RuleRequest struct {
	Rules map[string][]string `json:rules`
}

func NewRuleRequest() *RuleRequest {
	var rr RuleRequest
	rr.Rules = make(map[string][]string)
	return &rr
}

func printRules(rc *rule.RuleConfig) {
	fmt.Printf("%s", rc)
}

func getRules(host domain) (*rule.RuleConfig, error) {
	rule := rule.RuleConfig{}

	response, err := http.Get(fmt.Sprintf("http://%s/rules", host))
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &rule)
	if err != nil {
		return nil, err
	}

	return &rule, nil
}

func updateRules(host domain, blacklist, whitelist strArray) error {
	rc, err := getRules(host)
	if err != nil {
		return fmt.Errorf("could not get rules: %v", err)
	}

	rules := NewRuleRequest()
	if len(blacklist) > 0 {
		if rc.Blacklist != nil {
			blacklist.Merge(rc.Blacklist)
		}
		rules.Rules["blacklist"] = blacklist
	}

	if len(whitelist) > 0 {
		if rc.Whitelist != nil {
			whitelist.Merge(rc.Whitelist)
		}
		rules.Rules["whitelist"] = whitelist
	}

	body, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("could not build request: %v", err)
	}

	response, err := http.Post(
		fmt.Sprintf("http://%s/rules", host),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
		fmt.Printf("success\n")
	default:
		return fmt.Errorf("shit's broke capn'")
	}

	return nil
}

func main() {
	var whitelist strArray
	var blacklist strArray
	var host domain

	flag.Bool("overwrite", false, "replace rules, do not append")
	flag.Var(&whitelist, "whitelsist", "whitelist domain[s]")
	flag.Var(&blacklist, "blacklist", "blacklist domain[s]")
	flag.Var(&host, "host", "where to send the request")
	flag.Parse()

	if len(blacklist) > 0 || len(whitelist) > 0 {
		err := updateRules(host, blacklist, whitelist)
		if err != nil {
			fmt.Printf("could not update rules: %v\n", err)
		}
	} else {
		rc, err := getRules(host)
		if err != nil {
			fmt.Printf("could not get rules: %v\n", err)
		}

		printRules(rc)
	}
}
