package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/askhatbaialiev/WASA/service/api/reqcontext"
	"github.com/askhatbaialiev/WASA/service/database"
	"github.com/julienschmidt/httprouter"
)

var (
	errNotGroup  = errors.New("conversation is not a group")
	errNotMember = errors.New("user is not a member")
)

// POST /groups
// Body: {"name": "G", "members": ["<userId>", ...]}
// Creates a group conversation containing the creator plus the listed members.
func (rt *_router) createGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params, ctx reqcontext.RequestContext) {
	var req struct {
		Name    string   `json:"name"`
		Members []string `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "A group name is required")
		return
	}

	// Build a de-duplicated member set that always includes the creator.
	seen := map[string]bool{ctx.UserID: true}
	members := []string{ctx.UserID}
	for _, m := range req.Members {
		if m == "" || seen[m] {
			continue
		}
		if _, err := rt.db.GetUser(m); err != nil {
			continue // skip unknown users rather than failing the whole request
		}
		seen[m] = true
		members = append(members, m)
	}

	conv, err := rt.db.CreateConversation(req.Name, true, members)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(conv)
}

// GET /conversations/:conversationId/members — list a group's members.
func (rt *_router) getGroupMembers(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	ok, err := rt.db.IsMember(conversationID, ctx.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	members, err := rt.db.GetMembers(conversationID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(members)
}

// POST /conversations/:conversationId/members
// Body: {"userId": "..."} — add a user to a group (members only).
func (rt *_router) addToGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	if _, err := rt.requireGroupMember(w, conversationID, ctx.UserID); err != nil {
		return
	}

	var req struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "A userId is required")
		return
	}
	if _, err := rt.db.GetUser(req.UserID); err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	if err := rt.db.AddMember(conversationID, req.UserID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /conversations/:conversationId/members/me — leave a group.
// The conversation is deleted once its last member leaves.
func (rt *_router) leaveGroup(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	if _, err := rt.requireGroupMember(w, conversationID, ctx.UserID); err != nil {
		return
	}

	if err := rt.db.RemoveMember(conversationID, ctx.UserID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /conversations/:conversationId/name
// Body: {"name": "..."} — rename a group (members only).
func (rt *_router) setGroupName(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	if _, err := rt.requireGroupMember(w, conversationID, ctx.UserID); err != nil {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "A group name is required")
		return
	}

	if err := rt.db.SetConversationName(conversationID, req.Name); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PUT /conversations/:conversationId/photo
// Body: {"photo": "<base64 data URI>"} — set a group's picture (members only).
func (rt *_router) setGroupPhoto(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ctx reqcontext.RequestContext) {
	conversationID := ps.ByName("conversationId")

	if _, err := rt.requireGroupMember(w, conversationID, ctx.UserID); err != nil {
		return
	}

	var req struct {
		Photo string `json:"photo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := rt.db.SetConversationPhoto(conversationID, req.Photo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// requireGroupMember validates that the conversation exists, is a group, and the
// caller is a member. On failure it writes the proper status and returns an error.
func (rt *_router) requireGroupMember(w http.ResponseWriter, conversationID, userID string) (database.Conversation, error) {
	conv, err := rt.db.GetConversation(conversationID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return database.Conversation{}, err
	}
	if !conv.IsGroup {
		writeError(w, http.StatusBadRequest, "This operation is only allowed on groups")
		return database.Conversation{}, errNotGroup
	}
	ok, err := rt.db.IsMember(conversationID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return database.Conversation{}, err
	}
	if !ok {
		w.WriteHeader(http.StatusForbidden)
		return database.Conversation{}, errNotMember
	}
	return conv, nil
}
