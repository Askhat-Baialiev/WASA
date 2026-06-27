package api

import (
	"encoding/json"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/julienschmidt/httprouter"
)

// POST /conversations/:conversationId/messages
// Body: {"text": "...", "replyToMessageId": "..."} — both optional but text is currently required.
func (rt *_router) sendMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	if _, err := rt.db.GetConversation(conversationID); err != nil {
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

	// A message may carry text, an uploaded image (base64 data URI, not a URL),
	// or both — but at least one of them must be present.
	var req struct {
		Text             string `json:"text"`
		Photo            string `json:"photo"`
		ReplyToMessageID string `json:"replyToMessageId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Text == "" && req.Photo == "") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	message, err := rt.db.SendMessage(conversationID, ctx.UserID, req.Text, req.Photo, req.ReplyToMessageID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(message)
}

// DELETE /conversations/:conversationId/messages/:messageId
// Only the sender of the message can delete it.
func (rt *_router) deleteMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	messageID := ps.ByName("messageId")

	msg, err := rt.db.GetMessage(messageID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if msg.SenderID != ctx.UserID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := rt.db.DeleteMessage(messageID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /conversations/:conversationId/messages/:messageId/forward
// Body: {"targetConversationId": "..."}
// Forwards an existing message into another conversation. The forwarder must be
// a member of BOTH the source conversation (to see the message) and the target
// one — and the target must already exist (you can only forward into a
// preexisting chat). The copy is flagged "forwarded".
func (rt *_router) forwardMessage(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	sourceConversationID := ps.ByName("conversationId")
	messageID := ps.ByName("messageId")

	// The caller must be a member of the source conversation to forward from it.
	ok, err := rt.db.IsMember(sourceConversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	msg, err := rt.db.GetMessage(messageID)
	if err != nil || msg.ConversationID != sourceConversationID {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var req struct {
		TargetConversationID string `json:"targetConversationId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetConversationID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// The target chat must already exist and the caller must be a member of it.
	if _, err := rt.db.GetConversation(req.TargetConversationID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	ok, err = rt.db.IsMember(req.TargetConversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	forwarded, err := rt.db.ForwardMessage(messageID, req.TargetConversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(forwarded)
}
