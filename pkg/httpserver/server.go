package httpserver

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func SetupHTTPServer(responder Responder) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		responder.Metrics(w, r)
	})
	http.HandleFunc("/model", func(w http.ResponseWriter, r *http.Request) {
		responder.Model(w, r)
	})

	// TODO make this configurable - maybe even viperize it.
	http.ListenAndServe(":3000", nil)
	log.Info("Http server started!")
}
