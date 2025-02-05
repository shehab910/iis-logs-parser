package routes

import (
	"errors"
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

func handleCreateDomain(ctx *gin.Context) {
	var newDomain models.Domain

	if err := ctx.ShouldBindJSON(&newDomain); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Add defaults
	newDomain.IsActive = true
	newDomain.UserID = ctx.GetUint("userId")

	if err := newDomain.Validate(); err != nil {
		log.Error().Err(err).Msg("error binding json")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	res := db.GormDB.Create(&newDomain)

	if res.Error != nil {
		log.Error().Err(res.Error).Msg("error creating domain")
		ctx.JSON(500, gin.H{
			"error": res.Error.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message":  "Domain created successfully",
		"domainId": newDomain.ID,
	})
}

func handleReadDomains(ctx *gin.Context) {
	userId := ctx.GetUint("userId")

	var domains []models.Domain
	res := db.GormDB.Where("user_id = ?", userId).Find(&domains)

	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "No domains found",
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": res.Error.Error(),
		})
	}

	ctx.JSON(http.StatusOK, domains)
}

func handleUpdateDomain(ctx *gin.Context) {
	userId := ctx.GetUint("userId")
	domainId, err := strconv.ParseUint(ctx.Param("id"), 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid domain ID",
		})
		return
	}

	var domainUpdate models.DomainUpdateRequest
	if err := ctx.ShouldBindJSON(&domainUpdate); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var existingDomain models.Domain
	result := db.GormDB.First(&existingDomain, domainId)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}

	if existingDomain.UserID != userId {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to update this domain"})
		return
	}

	existingDomain.PartialUpdate(domainUpdate)

	if err := existingDomain.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if result := db.GormDB.Save(&existingDomain); result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update domain"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Domain updated successfully"})

}

func handleDeleteDomain(ctx *gin.Context) {
	userId := ctx.GetUint("userId")
	domainId, err := strconv.ParseUint(ctx.Param("id"), 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid domain ID",
		})
		return
	}

	var existingDomain models.Domain
	result := db.GormDB.First(&existingDomain, domainId)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}

	if existingDomain.UserID != userId {
		ctx.JSON(http.StatusForbidden, gin.H{"error": "You are not authorized to delete this domain"})
		return
	}

	// In spite of having a constraint set, .Select must be called here to cascade deleting, very weird behavior
	if result := db.GormDB.Select("LogFiles").Delete(&existingDomain); result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete domain"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Domain deleted successfully"})
}
