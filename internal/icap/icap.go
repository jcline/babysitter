package icap

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elico/icap"
	"github.com/rs/zerolog/log"
)

var istag uint64

func Start(address string, done *sync.WaitGroup) error {
	defer done.Done()

	atomic.StoreUint64(&istag, uint64(time.Now().Unix()))
	icap.HandleFunc("/", icapHandler)
	return icap.ListenAndServe(address, icap.HandlerFunc(icapHandler))
}

func IncrementTag() {
	atomic.AddUint64(&istag, 1)
}

func icapHandler(response icap.ResponseWriter, request *icap.Request) {
	start := time.Now()
	headers := response.Header()
	headers.Set("ISTag", fmt.Sprintf("\"%d\"", atomic.LoadUint64(&istag)))
	headers.Set("Service", "Babysitter v1.0")

	var status int
	var wrappedStatus int

	switch request.Method {
	case "OPTIONS":
		// We only support request modification with this server
		headers.Set("Methods", "REQMOD")
		headers.Set("Allow", "204")

		// How many connections do we permit the client to establish
		headers.Set("Max-Connections", "1024")

		// How long can the client cache the response
		headers.Set("Options-TTL", "3600")

		// We don't want the client sending us bodies currently
		headers.Set("Preview", "0")
		//headers.Set("Transfer-Ignore", "*")
		response.WriteHeader(http.StatusOK, nil, false)
		status = http.StatusOK
	case "REQMOD":
		headers.Set("Cache-Control", "no-cache")

		request.Request.Header.Add("Permitted", "no")

		status = http.StatusOK
		response.WriteHeader(status, request.Request, false)
		//status = http.StatusNoContent
	default:
		// wat, we only support OPTIONS and REQMOD
		response.WriteHeader(http.StatusMethodNotAllowed, nil, false)
		status = http.StatusMethodNotAllowed
	}

	duration := time.Now().Sub(start)

	event := log.Info().
		Str("method", request.Method).
		Stringer("url", request.URL).
		Int("status", status).
		Int("wrapped_status", wrappedStatus).
		Int("preview_size", len(request.Preview)).
		Str("proto_version", request.Proto).
		Dur("duration", duration).
		Str("remote_addr", request.RemoteAddr)
	if request.Request != nil {
		event.Str("domain", request.Request.Host).
			Str("client", request.Request.RemoteAddr)
	}

	event.Msg("")
}
