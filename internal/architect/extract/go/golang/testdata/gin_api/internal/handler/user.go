package handler

import (
	"github.com/example/ginapi/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func SetupRoutes(r *gin.Engine) {
	h := &UserHandler{}
	r.GET("/users", h.GetUsers)
	r.GET("/users/:id", h.GetUser)
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get all users"})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Get user by ID"})
}
