package middleware

import (
	"iis-logs-parser/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Authenticate(ctx *gin.Context) {
	tokenString := ctx.GetHeader("Authorization")

	if tokenString == "" {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	claims, err := utils.ValidateToken(tokenString)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		return
	}

	if isVerified, ok := claims["is_verified"].(bool); !isVerified || !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Please login again and verify your email",
		})
		return
	}

	userId, ok := claims["userId"].(float64)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Please login again and verify your email",
		})
		return
	}
	ctx.Set("userId", uint(userId))
	ctx.Set("role", claims["role"])
	ctx.Set("email", claims["email"])
	ctx.Next()
}
