// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package response

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// Upload describes an uploaded file
type Upload struct {
	UID      string `json:"uid"`
	FileName string `json:"file_name"`
	Size     int64  `json:"file_size"`
}

// UploadResponse describes and uploaded file, including user's JWT needed for
// later access.
type UploadResponse struct {
	Upload `json:"file"`
	Token  string `json:"token"`
}

// UseLog sets an external logger for use by this package.
func UseLog(_log *logrus.Logger) {
	log = _log
}

// WriteJSON writes the specified object as JSON, using the specified
// indentation string.
func WriteJSON(w http.ResponseWriter, obj interface{}, indent string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", indent)
	if err := encoder.Encode(obj); err != nil {
		log.Infof("JSON encode error: %v", err)
	}
}

// WritePlainText sets the Content-Type to text/plain and writes the string.
func WritePlainText(w http.ResponseWriter, str string) {
	writeText(w, "text/plain; charset=utf-8", str)
}

// WriteHTML sets the Content-Type to text/html and writes the string.
func WriteHTML(w http.ResponseWriter, str string) {
	writeText(w, "text/html; charset=utf-8", str)
}

func writeText(w http.ResponseWriter, contentType, str string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, str)
}

// SendFile attempts to transfer the specified file to the ResponseWriter.
func SendFile(w http.ResponseWriter, filePath string) error {
	// Attempt to open the specified file. Caller must sanitize.
	File, err := os.OpenFile(filePath, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}

	// Stat the file for size and base name
	stat, err := File.Stat()
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, stat.Name()))
	w.Header().Set("Content-Length", strconv.Itoa(int(stat.Size())))
	w.WriteHeader(http.StatusOK)

	// Use io.Copy so we do not have to load the entire file into memory.
	_, err = io.Copy(w, bufio.NewReader(File))
	return err
}
