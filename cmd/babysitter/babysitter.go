package main

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/jcline/babysitter/internal/icap"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	debug := flag.Bool("debug", false, "enables debug logging")
	verbose := flag.Bool("verbose", false, "enables verbose logging")
	listen := flag.String("listen", "localhost:9001", "address to listen on")
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

	log.Info().Str("address", *listen).Msg("starting babysitter")
	defer log.Info().Msg("stopping")

	done := &sync.WaitGroup{}

	done.Add(1)
	go icap.Start(*listen, done)

	done.Wait()
}
