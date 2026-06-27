package api

import (
	"encoding/json"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/askhatbaialiev/WASA/service/database"
	"github.com/julienschmidt/httprouter"
)

// conversationResponse is what GET /conversations/:id returns —
// the conversation header plus its full message history.
type conversationResponse struct {
	Conversation database.Conversation `json:"conversation"`
	Messages     []database.Message    `json:"messages"`
}

// GET /conversations — list every chat the logged-in user is part of.
func (rt *_router) getMyConversations(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	conversations, err := rt.db.GetConversations(ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(conversations)
}

// GET /conversations/:conversationId — chat header + all messages.
// Opening a chat also marks every message in it as "read" by the caller
// (this is what turns sender ticks blue, WhatsApp-style).
func (rt *_router) getConversation(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	conv, err := rt.db.GetConversation(conversationID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ok, err := rt.db.IsMember(conversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Mark everything in this chat as read for the current user (best-effort).
	_ = rt.db.MarkConversationRead(conversationID, ctx.UserID)

	messages, err := rt.db.GetMessages(conversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(conversationResponse{
		Conversation: conv,
		Messages:     messages,
	})
}

// POST /conversations — start a direct (1:1) conversation with another user.
// Body: {"userId": "<other-user-id>"}.
// If the DM already exists, returns it (200) instead of creating a duplicate.
func (rt *_router) createDirectConversation(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Make sure the target user actually exists.
	if _, err := rt.db.GetUser(req.UserID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Reuse the existing DM if one is already there.
	if existing, err := rt.db.FindDirectConversation(ctx.UserID, req.UserID); err == nil {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(existing)
		return
	}

	conv, err := rt.db.CreateConversation("", false, []string{ctx.UserID, req.UserID})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// For a DM, show the other person's name as the chat name.
	if other, err := rt.db.GetUser(req.UserID); err == nil {
		conv.Name = other.Name
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(conv)
}
