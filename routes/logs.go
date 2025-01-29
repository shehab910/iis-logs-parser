package routes

import (
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

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

func handleUploadFile(c *gin.Context) {
	log.Info().Msg("Trying to get file")
	file, err := c.FormFile("logfile")

	if err != nil {
		log.Error().Err(err).Msg("Failed to get file")
		c.JSON(400, gin.H{
			"message": "No file uploaded",
		})
		return
	}
	log.Info().Msg("File uploaded")

	filename := filepath.Base(file.Filename)

	logFileEntry := models.LogFile{
		Name:   filename,
		Size:   uint(file.Size),
		Status: models.StatusPending,
	}

	tx := db.GormDB.Begin()
	res := tx.Create(&logFileEntry)

	if res.Error != nil {
		log.Err(res.Error).Msg("Couldn't create log file entry in db")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save db entry",
		})
		return
	}
	savedFileName := filepath.Join("uploaded_logs", filename+"-"+strconv.FormatUint(uint64(logFileEntry.ID), 10))
	err = c.SaveUploadedFile(file, savedFileName)
	if err != nil {
		log.Err(err).Msg("Couldn't save file")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save file",
		})
		err = tx.Rollback().Error
		if err != nil {
			log.Err(err).Msg("Couldn't rollback transaction")
		}
		return
	}
	log.Info().Msg("File Saved")

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
