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
	"strings"

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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "couldn't parse multipartform", err)
		return
	}
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form", err)
		return
	}

	fileType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileType)
	if err != nil || (mediaType != "image/jpeg" && mediaType != "image/png") {
		respondWithError(w, http.StatusBadRequest, "invalid file format", err)
		return
	}
	splitFileType := strings.Split(fileType, "/")
	fileExtension := splitFileType[1]
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	randomStr := base64.RawURLEncoding.EncodeToString(randomBytes)
	uniquePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%v.%v", randomStr, fileExtension))
	f, err := os.Create(uniquePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create file path", err)
		return
	}
	defer f.Close()
	_, err = io.Copy(f, file)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to copy file contents", err)
		return
	}

	videoMetaData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "unable to get video from db", err)
		return
	}

	if videoMetaData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "unauthorized video user", err)
		return
	}

	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%v.%v", cfg.port, randomStr, fileExtension)

	videoMetaData.ThumbnailURL = &thumbnailURL

	cfg.db.UpdateVideo(videoMetaData)

	respondWithJSON(w, http.StatusOK, videoMetaData)
}
