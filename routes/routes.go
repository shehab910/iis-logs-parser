package routes

import (
	"iis-logs-parser/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server *gin.Engine) {
	v1 := server.Group("/api/v1")
	{
		authenticated := v1.Group("/")
		authenticated.Use(middleware.Authenticate)
		{
			authenticated.GET("/example", handleExampleAuth)
		}
		v1.POST("/log-entries")       //, CreateLogEntry)
		v1.GET("/log-entries/:id")    //, GetLogEntry)
		v1.PUT("/log-entries/:id")    //, UpdateLogEntry)
		v1.DELETE("/log-entries/:id") //, DeleteLogEntry)

		v1.POST("/upload-logs", handleUploadFile)

		users := v1.Group("/users")
		{
			users.POST("/register", handleRegisterUser)
			users.POST("/login", handleLoginUser)
			users.GET("/verify", handleVerifyUser)
		}
	}

}
