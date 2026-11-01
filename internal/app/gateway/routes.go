package gateway

import "net/http"

func (a *App) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/notifications", a.handler.handleSend)
	mux.HandleFunc("GET /v1/notifications/{id}", a.handler.handleGet)
}
