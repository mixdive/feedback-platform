package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatusResponse struct {
	Message string `json:"message" validate:"required"` //Message
	Code    int    `json:"code" validate:"required"`    //Code of status which is always 200
} //@name StatusResponse

func Status(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, &StatusResponse{
		Message: msg,
		Code:    http.StatusOK,
	})
}

func Success(c *gin.Context, response any) {
	c.JSON(http.StatusOK, response)
}

// Created responds with HTTP 201. Use it for POSTs that produce a new resource.
func Created(c *gin.Context, response any) {
	c.JSON(http.StatusCreated, response)
}

// NoContent responds with HTTP 204 — for successful operations that have
// nothing to return (logout, soft-delete, archive).
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
