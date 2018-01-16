package httpserver

import "net/http"

type Responder interface {
	Metrics(w http.ResponseWriter, r *http.Request)
	Model(w http.ResponseWriter, r *http.Request)
}
