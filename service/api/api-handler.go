package api

import "net/http"

func (rt *_router) Handler() http.Handler {
	// Public — no authentication
	rt.router.POST("/session", rt.doLogin)
	rt.router.GET("/liveness", rt.liveness)

	// Authenticated — Bearer token required (rt.wrap)
	rt.router.GET("/users", rt.wrap(rt.getUsers))
	rt.router.PUT("/users/me/username", rt.wrap(rt.setMyUsername))
	rt.router.PUT("/users/me/photo", rt.wrap(rt.setMyPhoto))

	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))
	rt.router.POST("/conversations", rt.wrap(rt.createDirectConversation))
	rt.router.GET("/conversations/:conversationId", rt.wrap(rt.getConversation))

	rt.router.POST("/groups", rt.wrap(rt.createGroup))
	rt.router.GET("/conversations/:conversationId/members", rt.wrap(rt.getGroupMembers))
	rt.router.POST("/conversations/:conversationId/members", rt.wrap(rt.addToGroup))
	rt.router.DELETE("/conversations/:conversationId/members/me", rt.wrap(rt.leaveGroup))
	rt.router.PUT("/conversations/:conversationId/name", rt.wrap(rt.setGroupName))
	rt.router.PUT("/conversations/:conversationId/photo", rt.wrap(rt.setGroupPhoto))

	rt.router.POST("/conversations/:conversationId/messages", rt.wrap(rt.sendMessage))
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId", rt.wrap(rt.deleteMessage))
	rt.router.POST("/conversations/:conversationId/messages/:messageId/forward", rt.wrap(rt.forwardMessage))

	rt.router.PUT("/conversations/:conversationId/messages/:messageId/reaction", rt.wrap(rt.addReaction))
	rt.router.DELETE("/conversations/:conversationId/messages/:messageId/reaction", rt.wrap(rt.removeReaction))

	return rt.router
}
