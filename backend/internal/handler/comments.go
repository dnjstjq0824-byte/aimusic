package handler

import (
	"github.com/gin-gonic/gin"

	"avibe-platform/backend/internal/domain"
	"avibe-platform/backend/internal/service"
)

// CommentHandler serves listener comments on tracks.
type CommentHandler struct {
	Comments *service.CommentService
}

// List returns newest-first comments for a READY track (public).
//
//	GET /tracks/:id/comments?before=<RFC3339Nano>&limit=
func (h *CommentHandler) List(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	page, err := h.Comments.List(c.Request.Context(), id, c.Query("before"), queryInt(c, "limit", 20))
	if err != nil {
		AbortError(c, err)
		return
	}
	ok(c, page)
}

type commentRequest struct {
	Body string `json:"body" binding:"required"`
}

// Create posts a comment (login required).
//
//	POST /tracks/:id/comments {body}
func (h *CommentHandler) Create(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		badID(c)
		return
	}
	var req commentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abort(c, 400, "ERR_INVALID_INPUT", err.Error())
		return
	}
	cm, err := h.Comments.Create(c.Request.Context(), CurrentUserID(c), id, req.Body)
	if err != nil {
		AbortError(c, err)
		return
	}
	created(c, cm)
}

// Delete removes a comment; its author or an admin.
//
//	DELETE /tracks/:id/comments/:commentId
func (h *CommentHandler) Delete(c *gin.Context) {
	id, cid := c.Param("id"), c.Param("commentId")
	if !validUUID(id) || !validUUID(cid) {
		badID(c)
		return
	}
	isAdmin := CurrentRole(c) == domain.RoleAdmin
	if err := h.Comments.Delete(c.Request.Context(), CurrentUserID(c), isAdmin, id, cid); err != nil {
		AbortError(c, err)
		return
	}
	ok(c, gin.H{"deleted": true})
}
