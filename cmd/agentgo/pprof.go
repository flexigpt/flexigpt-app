//go:build pprof

package main

import (
	"log"
	"net/http"

	//nolint:gosec // Side-effect import registers pprof endpoints.
	_ "net/http/pprof"
)

func init() {
	startPprof(true)
}

func startPprof(enabled bool) {
	if !enabled {
		return
	}

	go func() {
		// go tool pprof -http=:8080 http://127.0.0.1:6060/debug/pprof/profile\?seconds\=10
		log.Println("starting pprof server on http://127.0.0.1:6060")

		//nolint:gosec // Ok.
		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
}
