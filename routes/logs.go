package routes

import (
	"errors"
	"fmt"
	db "iis-logs-parser/database"
	"iis-logs-parser/models"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	FileID  uint   `json:"file_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

func saveFile(file *multipart.FileHeader, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}

func processFile(tx *gorm.DB, file *multipart.FileHeader, domainId int64) (UploadResponse, error) {
	filename := filepath.Base(file.Filename)

	logFileEntry := models.LogFile{
		Name:     filename,
		Size:     uint(file.Size),
		Status:   models.StatusPending,
		DomainID: uint(domainId),
	}

	if err := tx.Create(&logFileEntry).Error; err != nil {
		return UploadResponse{
			Success: false,
			Message: "Failed to save database entry",
			Error:   err.Error(),
		}, err
	}

	savedFileName := filepath.Join("uploaded_logs", filename+"-"+strconv.FormatUint(uint64(logFileEntry.ID), 10))
	if err := saveFile(file, savedFileName); err != nil {
		return UploadResponse{
			Success: false,
			Message: "Failed to save file",
			Error:   err.Error(),
		}, err
	}

	return UploadResponse{
		Success: true,
		Message: "File uploaded & saved",
		FileID:  logFileEntry.ID,
	}, nil
}

func handleUploadLogFiles(ctx *gin.Context) {
	userId := ctx.GetUint("userId")
	domainStr := ctx.PostForm("domain")

	// Validate and parse domainId
	domainId, err := strconv.ParseInt(domainStr, 10, 64)
	if err != nil {
		log.Error().Err(err).Msg("Invalid domain")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid domain",
		})
		return
	}
	log.Info().Uint("userId", userId).Int64("userEnteredDomainId", domainId).Msg("Checking user domain ownership")
	res := db.GormDB.First(&models.Domain{}, "id = ? AND user_id = ?", domainId, userId)
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrRecordNotFound) {
			log.Err(res.Error).Msg("Domain not found for user")
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Domain not found",
			})
			return
		}
		log.Err(res.Error).Msg("Error finding domain")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error finding domain",
		})
		return
	}
	log.Info().Uint("userId", userId).Int64("userEnteredDomainId", domainId).Msg("Domain found")
	////

	// Get uploaded files
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "No files uploaded",
		})
	}
	files := form.File["logfiles"]
	////

	// Process files
	responses := make([]UploadResponse, 0, len(files))
	tx := db.GormDB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Error().Interface("recover", r).Msg("Transaction rolled back due to panic")
		}
	}()

	for _, file := range files {
		response, err := processFile(tx, file, domainId)

		responses = append(responses, response)
		if err != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error":     "Couldn't save db entry",
				"responses": responses,
			})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Err(err).Msg("Couldn't commit transaction")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't save db entry",
		})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"message":   "File upload completed",
		"responses": responses,
	})
}

func handleGetAllLogFilesForUser(ctx *gin.Context) {
	userId := ctx.GetUint("userId")
	log.Warn().Uint("userId", userId).Msg("userId")

	var logFiles []models.LogFile
	err := db.GormDB.Joins("JOIN domains ON domains.id = log_files.domain_id").Where("domains.user_id = ?", userId).Find(&logFiles).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't get log files",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"result": logFiles,
	})
}

func handleGetDomainLogFiles(ctx *gin.Context) {
	domainId := ctx.Param("id")
	userId := ctx.GetUint("userId")

	var logFiles []models.LogFile
	err := db.GormDB.Joins("JOIN domains ON domains.id = log_files.domain_id").Where("domains.user_id = ? AND domains.id = ?", userId, domainId).Find(&logFiles).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Couldn't get log files",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"result": logFiles,
	})
}

func handleDeleteLogFile(ctx *gin.Context) {
	fileId := ctx.Param("id")
	userId := ctx.GetUint("userId")

	var logFile models.LogFile
	err := db.GormDB.Joins("JOIN domains ON domains.id = log_files.domain_id").Where("domains.user_id = ?", userId).First(&logFile, fileId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Log file not found",
			})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Something went wrong",
		})
		return
	}

	err = db.GormDB.Select("LogEntries").Delete(&logFile).Error
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Couldn't delete log file",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "File deleted",
	})
}
