package routes

import (
	"iis-logs-parser/config"
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"iis-logs-parser/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func handleRegisterUser(ctx *gin.Context) {
	var user models.User

	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	//Add defaults
	user.Role = "user"
	user.Verified = false

	if err := user.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	hashedPass, err := utils.HashPassword(user.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong!",
		})
		return
	}
	user.Password = hashedPass

	res := db.GormDB.Create(&user)

	if res.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save user",
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message": "User created, please login to verify your email",
		"email":   user.Email,
	})
}

func handleLoginUser(ctx *gin.Context) {
	var user = struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}{}

	if err := ctx.ShouldBindJSON(&user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := utils.Validate.Struct(user); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var dbUser models.User
	res := db.GormDB.Where("email = ?", user.Email).First(&dbUser)

	if res.Error != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid Credentials",
		})
		return
	}

	if !utils.CheckPasswordHash(user.Password, dbUser.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid Credentials",
		})
		return
	}

	token, err := utils.GenerateToken(dbUser.Email, dbUser.ID, dbUser.Role, dbUser.Verified)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't generate token",
		})
		return
	}

	if !dbUser.Verified {
		isLastSentTokenExpired := time.Since(dbUser.LastLoginAt) > time.Minute*config.TOKEN_EXPIRATION_MINS
		userNeverLoggedIn := dbUser.LastLoginAt.IsZero()

		if isLastSentTokenExpired || userNeverLoggedIn {
			err := utils.SendVerifyUserEmail(dbUser.Email, token)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": "Couldn't send verification email",
				})
				return
			}
			dbUser.LastLoginAt = time.Now()
			db.GormDB.Save(&dbUser)
		}
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Please verify your email, verification link sent",
		})
		return
	}

	dbUser.LastLoginAt = time.Now()
	db.GormDB.Save(&dbUser)
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Logged in",
		"token":   token,
	})

}

func handleVerifyUser(ctx *gin.Context) {
	token := ctx.Query("token")
	claims, err := utils.ValidateToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token, Login again & reverify your email",
		})
		return
	}

	userId := uint(claims["userId"].(float64))
	var user models.User
	res := db.GormDB.First(&user, userId)

	if res.Error != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	if user.Verified {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "User already verified, try logging in again",
		})
		return
	}

	user.Verified = true
	db.GormDB.Save(&user)

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User verified, please login to renew your token",
	})
}
