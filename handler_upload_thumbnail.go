package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// upload implementation
	const MaxMemory = 10 << 20 // 10 MB
	err = r.ParseMultipartForm(MaxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get thumbnail file from form", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse media type", err)
		return
	}

	var ext string
	switch mediaType {
	case "image/jpeg":
		ext = "jpg"
	case "image/png":
		ext = "png"
	default:
		respondWithError(w, http.StatusBadRequest, "Unsupported media type", nil)
		return
	}

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get video data", err)
		return
	}

	if videoData.UserID != userID {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	thumbnailId := make([]byte, 32)
	rand.Read(thumbnailId)
	encodedThumbnailId := base64.RawURLEncoding.EncodeToString(thumbnailId)

	assetPath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", encodedThumbnailId, ext))
	dstFile, err := os.Create(assetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create thumbnail file", err)
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write thumbnail file", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, encodedThumbnailId, ext)
	videoData.ThumbnailURL = &thumbnailURL
	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video data", err)
		return
	}

	updatedVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get updated video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, updatedVideo)
}
