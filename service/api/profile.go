package api

import (
	"encoding/json"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// writeError sends a JSON {"error": "..."} body with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// PUT /users/me/username
// Body: {"name": "..."}
// Changes the caller's display name. Names are unique, so trying to take a name
// that another user already owns fails with 409 and a clear error message.
func (rt *_router) setMyUserName(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if len(req.Name) < 3 || len(req.Name) > 16 {
		writeError(w, http.StatusBadRequest, "Name must be between 3 and 16 characters")
		return
	}

	// Reject names already taken by a *different* user (re-saving your own name is fine).
	if existing, err := rt.db.GetUserByName(req.Name); err == nil && existing.ID != ctx.UserID {
		writeError(w, http.StatusConflict, "That name is already taken by another user")
		return
	}

	if err := rt.db.SetUserName(ctx.UserID, req.Name); err != nil {
		// The UNIQUE constraint is also enforced at the DB level as a safety net.
		writeError(w, http.StatusConflict, "That name is already taken by another user")
		return
	}

	user, err := rt.db.GetUser(ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}

// PUT /users/me/photo
// Body: {"photo": "<base64 data URI>"}
func (rt *_router) setMyPhoto(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var req struct {
		Photo string `json:"photo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := rt.db.SetUserPhoto(ctx.UserID, req.Photo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := rt.db.GetUser(ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(user)
}
