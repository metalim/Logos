//go:build !js

package game

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func startProfilingServer() {
	go func() {
		log.Printf("profile: pprof http://%s/debug/pprof/", profilingAddr)
		if err := http.ListenAndServe(profilingAddr, nil); err != nil {
			log.Printf("profile: pprof server: %v", err)
		}
	}()
}
