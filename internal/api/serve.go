package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/jcline/babysitter/internal/rule"
)

func Start(address string) error {

	chain := alice.New().
		Append(hlog.NewHandler(log.Logger)).
		Append(hlog.AccessHandler(
			func(
				r *http.Request,
				status, size int,
				duration time.Duration,
			) {
				hlog.FromRequest(r).Info().
					Str("method", r.Method).
					Stringer("url", r.URL).
					Int("status", status).
					Int("size", size).
					Dur("duration", duration).
					Msg("")
			},
		)).
		Append(hlog.RemoteAddrHandler("ip")).
		Append(hlog.UserAgentHandler("user_agent")).
		Append(hlog.RefererHandler("referer")).
		Append(hlog.RequestIDHandler("req_id", "Request-Id")).
		Then(http.HandlerFunc(ruleHandler))

	mux := http.NewServeMux()
	mux.Handle("/rules", chain)
	return http.ListenAndServe(address, mux)
}

func getRulesHandler(response http.ResponseWriter, request *http.Request) {
	type RuleResponse struct {
		rules map[string][]string
	}
	rules := rule.RuleManager.GetRules()

	body, err := json.Marshal(rules)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.WriteHeader(http.StatusOK)
	b, err := response.Write(body)
	if b != len(body) || err != nil {
		hlog.FromRequest(request).Error().
			Int("written", b).
			Int("expected", len(body)).
			Err(err).
			Msg("writing failed")
		return
	}
}

func updateRulesHandler(response http.ResponseWriter, request *http.Request) {
	type RuleUpdate struct {
		Rules map[string][]string
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		hlog.FromRequest(request).Error().
			Err(err).
			Msg("could not read update request body")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	var update RuleUpdate
	err = json.Unmarshal(body, &update)
	if err != nil {
		hlog.FromRequest(request).Error().
			Err(err).
			Msg("could not deserialize update request body")
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	rc, err := rule.NewRuleConfigFromMap(update.Rules)
	if err != nil {
		hlog.FromRequest(request).Error().
			Err(err).
			Msg("could not create rule config")
		response.WriteHeader(http.StatusBadRequest)
	}

	err = rule.RuleManager.Update(rc)
	if err != nil {
		hlog.FromRequest(request).Error().
			Err(err).
			Msg("could not create rule config")
		response.WriteHeader(http.StatusBadRequest)
	}

	response.WriteHeader(http.StatusOK)
	b, err := response.Write(body)
	if b != len(body) || err != nil {
		hlog.FromRequest(request).Error().
			Int("written", b).
			Int("expected", len(body)).
			Err(err).
			Msg("writing failed")
		return
	}
}

func ruleHandler(response http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case "GET":
		getRulesHandler(response, request)
	case "POST":
		updateRulesHandler(response, request)
	default:
		response.WriteHeader(http.StatusBadRequest)
	}
}
