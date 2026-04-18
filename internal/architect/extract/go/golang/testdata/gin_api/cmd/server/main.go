package main

import (
	"github.com/example/ginapi/internal/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	handler.SetupRoutes(r)
	r.Run(":8080")
}
