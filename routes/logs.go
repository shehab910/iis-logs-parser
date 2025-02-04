package routes

import (
	"errors"
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Function for testing
func handleExampleAuth(ctx *gin.Context) {
	userId := ctx.GetUint("userId")
	email := ctx.GetString("email")
	role := ctx.GetString("role")

	ctx.JSON(http.StatusOK, gin.H{
		"message": "You are authenticated",
		"userId":  userId,
		"email":   email,
		"role":    role,
	})
}

func handleUploadLogFile(c *gin.Context) {
	userId := c.GetUint("userId")
	log.Info().Msg("Trying to get file")
	file, form1Err := c.FormFile("logfile")
	domain, form2Err := strconv.ParseInt(c.PostForm("domain"), 10, 64)

	if form1Err != nil {
		log.Error().Err(form1Err).Msg("Failed to get file")
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "No file uploaded",
		})
		return
	}

	if form2Err != nil {
		log.Error().Err(form2Err).Msg("No domain provided")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No domain provided",
		})
		return
	}

	log.Info().Uint("userId", userId).Msg("File uploaded")
	log.Info().Uint("userId", userId).Int64("userEnteredDomainId", domain).Msg("Checking user domain ownership")
	res := db.GormDB.First(&models.Domain{}, "id = ? AND user_id = ?", domain, userId)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Err(res.Error).Msg("Domain not found for user")
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Domain not found",
			})
			return
		}
		log.Err(res.Error).Msg("Error finding domain")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error finding domain",
		})
		return
	}
	log.Info().Uint("userId", userId).Int64("userEnteredDomainId", domain).Msg("Domain found")

	log.Info().Uint("userId", userId).Int64("domain", domain).Msg("Saving log file entry in db")
	filename := filepath.Base(file.Filename)

	logFileEntry := models.LogFile{
		Name:     filename,
		Size:     uint(file.Size),
		Status:   models.StatusPending,
		DomainID: uint(domain),
	}

	tx := db.GormDB.Begin()
	res = tx.Create(&logFileEntry)

	if res.Error != nil {
		log.Err(res.Error).Uint("userId", userId).Int64("domain", domain).Msg("Couldn't create log file entry in db")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save db entry",
		})
		return
	}
	log.Info().Uint("userId", userId).Int64("domain", domain).Msg("Log file entry saved in db")

	log.Info().Uint("userId", userId).Int64("domain", domain).Msg("Saving log file in filesystem")
	savedFileName := filepath.Join("uploaded_logs", filename+"-"+strconv.FormatUint(uint64(logFileEntry.ID), 10))
	err := c.SaveUploadedFile(file, savedFileName)
	if err != nil {
		log.Err(err).Uint("userId", userId).Int64("domain", domain).Msg("Couldn't save file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save file",
		})
		err = tx.Rollback().Error
		if err != nil {
			log.Err(err).Msg("Couldn't rollback transaction")
		}
		return
	}
	log.Info().Uint("userId", userId).Int64("domain", domain).Msg("Saved log file in filesystem")

	err = tx.Commit().Error
	if err != nil {
		log.Err(err).Msg("Couldn't commit transaction")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save db entry",
		})
		err = os.Remove(savedFileName)
		if err != nil {
			log.Err(err).Msg("Couldn't remove file: " + savedFileName)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "File uploaded & saved",
		"file_id": logFileEntry.ID,
	})
}

// Function for testing
func GetLogFile(c *gin.Context) {
	paramID := c.Query("id")

	var logfileWithEntries models.LogFile
	// db.Model(&User{}).Preload("CreditCards").Find(&users)
	err := db.GormDB.Model(&models.LogFile{}).Preload("LogEntries").First(&logfileWithEntries, "id = ?", paramID)
	// err := db.GormDB.First(&logfileWithEntries, "id = ?", paramID)
	if err.Error != nil {
		log.Err(err.Error).Msg("Couldn't find log file")
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"result": logfileWithEntries,
	})
}
