package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (rt *_router) liveness(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusNoContent)
}
