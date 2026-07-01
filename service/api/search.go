package api

import (
	"encoding/json"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// GET /users?search=<query>
// Returns users whose name contains the query (case-insensitive LIKE).
// Used by the UI when starting a new direct conversation.
func (rt *_router) searchUsers(w http.ResponseWriter, r *http.Request, _ httprouter.Params, _ reqcontext.RequestContext) {
	query := r.URL.Query().Get("search")
	if query == "" {
		query = r.URL.Query().Get("name")
	}

	users, err := rt.db.SearchUsers(query)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(users)
}
