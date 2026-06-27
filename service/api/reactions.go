package api

import (
	"encoding/json"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// PUT /conversations/:conversationId/messages/:messageId/reaction
// Body: {"emoticon": "👍"}
// Sets the caller's single emoji reaction on the message (replacing any previous
// one — a user may have at most one reaction per message). The response carries
// the reaction's author name.
func (rt *_router) addReaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	ok, err := rt.db.IsMember(conversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	msg, err := rt.db.GetMessage(messageID)
	if err != nil || msg.ConversationID != conversationID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var req struct {
		Emoticon string `json:"emoticon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Emoticon == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	comment, err := rt.db.AddReaction(messageID, ctx.UserID, req.Emoticon)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(comment)
}

// DELETE /conversations/:conversationId/messages/:messageId/reaction
// Removes the caller's reaction on the message (if any).
func (rt *_router) removeReaction(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	ok, err := rt.db.IsMember(conversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := rt.db.RemoveReaction(messageID, ctx.UserID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
