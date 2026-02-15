package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	var result struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "", err
	}
	if len(result.Streams) == 0 {
		return "", fmt.Errorf("no streams found in %s", filePath)
	}

	width := result.Streams[0].Width
	height := result.Streams[0].Height

	ratio := float64(width) / float64(height)
	const tolerance = 0.05
	if ratio > (16.0/9.0)-tolerance && ratio < (16.0/9.0)+tolerance {
		return "16:9", nil
	}
	if ratio > (9.0/16.0)-tolerance && ratio < (9.0/16.0)+tolerance {
		return "9:16", nil
	}
	return "other", nil
}
