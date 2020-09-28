package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jcline/babysitter/internal/api"
	"github.com/jcline/babysitter/internal/icap"
	"github.com/jcline/babysitter/internal/rule"
)

func main() {
	var err error
	debug := flag.Bool("debug", false, "enables debug logging")
	verbose := flag.Bool("verbose", false, "enables verbose logging")
	listen := flag.String("listen", "localhost:9001", "address to listen on")
	apiListen := flag.String("apilisten", "localhost:80", "address to listen on")
	whitelist := flag.String(
		"whitelist",
		"/etc/babysitter/whitelist",
		"the file containing the domain whitelist")
	blacklist := flag.String(
		"blacklist",
		"/etc/babysitter/blacklist",
		"the file containing the domain blacklist")
	flag.Parse()

	if isatty.IsTerminal(os.Stdout.Fd()) {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	}

	log.Logger = log.With().Caller().Logger()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if *debug {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	rc, err := rule.NewRuleConfig(*whitelist, *blacklist)
	if err != nil {
		log.Error().Err(err).Msg("could not load rule config")
		os.Exit(1)
	}

	rule.RuleManager.Update(rc)

	log.Info().Str("address", *listen).Msg("starting babysitter")
	defer log.Info().Msg("stopping")

	done := &sync.WaitGroup{}

	done.Add(1)
	go icap.Start(*listen, done)
	go api.Start(*apiListen)

	done.Wait()
}
