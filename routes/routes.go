package routes

import "github.com/gin-gonic/gin"

func RegisterRoutes(server *gin.Engine) {
	v1 := server.Group("/api/v1")
	{
		v1.POST("/upload-logs", HandleUploadFile)
	}
}
