//go:build pprof

package main

import (
	"log"
	"net/http"

	// Side-effect import registers pprof endpoints.
	_ "net/http/pprof"
)

init() {
	startPprof(true)
}

func startPprof(enabled bool) {
	if !enabled {
		return
	}

	go func() {
		// go tool pprof -http=:8080 http://127.0.0.1:6060/debug/pprof/profile\?seconds\=10
		log.Println("Starting pprof server on http://127.0.0.1:6060")

		if err := http.ListenAndServe("127.0.0.1:6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
}
