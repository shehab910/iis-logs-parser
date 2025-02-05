package routes

import (
	"iis-logs-parser/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(server *gin.Engine) {
	v1 := server.Group("/api/v1")
	{
		usersV1 := v1.Group("/users")
		{
			usersV1.POST("/register", handleRegisterUser)
			usersV1.POST("/login", handleLoginUser)
			usersV1.GET("/verify", handleVerifyUser)
		}

		domainsV1 := v1.Group("/domains")
		domainsV1.Use(middleware.Authenticate)
		{
			domainsV1.GET("/", handleReadDomains)
			domainsV1.POST("/", handleCreateDomain)
			domainsV1.PUT("/:id", handleUpdateDomain)
			domainsV1.DELETE("/:id", handleDeleteDomain)
		}

		logsV1 := v1.Group("/logs")
		logsV1.Use(middleware.Authenticate)
		{
			logsV1.GET("/", handleGetAllLogFilesForUser)
			logsV1.GET("/domain/:id", handleGetDomainLogFiles)
			logsV1.POST("/upload", handleUploadLogFiles)
			logsV1.DELETE("/:id", handleDeleteLogFile)
		}
	}

}
