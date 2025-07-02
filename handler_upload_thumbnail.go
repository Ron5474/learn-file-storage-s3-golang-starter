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
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not upload image", err)
		return
	}

	mediaType := header.Header.Get("Content-Type")

	parsedType, _, err  := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err)
		return
	}
	if parsedType != "image/png" && parsedType != "image/jpg" {
		respondWithError(w, http.StatusBadRequest, "Invalid Image Format", nil)
		return
	}
	// imageData, err := io.ReadAll(file)

	// encodedImage := base64.StdEncoding.EncodeToString(imageData)

	// dataURL := fmt.Sprintf("data:%v;base64,%v", mediaType, encodedImage)

	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Unable to locate Video", err)
		return
	}

	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	fileExtenstion := strings.Split(mediaType, "/")[1]
	fileNameBytes := make([]byte, 32)
	_, err = rand.Read(fileNameBytes)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err)
		return
	}

	fileName := base64.RawURLEncoding.EncodeToString(fileNameBytes)

	storagePath := filepath.Join(cfg.assetsRoot, fileName) + "." + fileExtenstion
	imageFile, err := os.Create(storagePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err)
		return
	}
	io.Copy(imageFile, file)

	dataURL := fmt.Sprintf("http://localhost:%v/assets/%v.%v", cfg.port, fileName, fileExtenstion)
	metadata.ThumbnailURL = &dataURL

	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err)
		return
	}


	respondWithJSON(w, http.StatusOK, metadata)
}
