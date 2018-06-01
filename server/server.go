// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OneOfOne/xxhash"
	"github.com/chappjc/webfiles/response"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func UseLog(_log *logrus.Logger) {
	log = _log
}

const (
	defaultFilesPath = "uploads"
)

type Server struct {
	CookieStore *sessions.CookieStore
	SigningKey  string
	MaxFileSize int64
	FilesPath   string
}

func NewServer(secret string, maxFileSize int64) *Server {
	shaSum := sha256.Sum256([]byte(secret))
	server := &Server{
		SigningKey:  secret,
		CookieStore: sessions.NewCookieStore(shaSum[:]),
		MaxFileSize: maxFileSize,
		FilesPath:   defaultFilesPath,
	}

	server.CookieStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	}

	return server
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "webfiles is running!")
}

func (s *Server) File(w http.ResponseWriter, r *http.Request) {
	// get file id
	// check token
	// serve file
}

func (s *Server) FileList(w http.ResponseWriter, r *http.Request) {
	// check token
	// serve file list
}

func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			response.Error(w, http.StatusUnsupportedMediaType, "file upload Content-Type request must be multipart/form-data")
			return
		}
		log.Debugln(params)

		if strings.HasPrefix(mediaType, "multipart/") {
			mpFile, fileHeader, err := s.formFile("fileupload", r)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer mpFile.Close()

			// compute UID and write file
			hasher := xxhash.New64()
			numBytes, err := io.Copy(hasher, mpFile)
			if err != nil {
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			UID := fmt.Sprintf("%016x", hasher.Sum64())
			log.Infof("Hashed %d bytes. UID: %s", numBytes, UID)

			_, err = mpFile.Seek(0, io.SeekStart)
			if err != nil {
				log.Errorln(err)
			}

			fullPath, _ := filepath.Abs(filepath.Join(s.FilesPath, UID))
			if err = os.MkdirAll(fullPath, 0755); err != nil {
				log.Errorln(err)
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}

			fullFile := filepath.Join(fullPath, fileHeader.Filename)
			fid, err := os.OpenFile(fullFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				log.Errorln(err)
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer fid.Close()

			numBytes, err = io.Copy(fid, mpFile)
			if err != nil {
				log.Errorln(err)
				response.Error(w, http.StatusInternalServerError, err.Error())
				return
			}
			log.Infof("Wrote %d bytes: %s", numBytes, fullFile)

			resp := &response.Upload{
				UID:      UID,
				FileName: fileHeader.Filename,
				Size:     numBytes,
			}
			response.WriteJSON(w, resp, "    ")
		}

	default:
		response.Error(w, 400, "must be POST")
		return
	}
}

var multipartByReader = &multipart.Form{
	Value: make(map[string][]string),
	File:  make(map[string][]*multipart.FileHeader),
}

func (s *Server) formFile(key string, r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	if r.MultipartForm == multipartByReader {
		return nil, nil, errors.New("http: multipart handled by MultipartReader")
	}
	if r.MultipartForm == nil {
		err := r.ParseMultipartForm(s.MaxFileSize)
		if err != nil {
			return nil, nil, err
		}
	}
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if fhs := r.MultipartForm.File[key]; len(fhs) > 0 {
			f, err := fhs[0].Open()
			return f, fhs[0], err
		}
	}
	return nil, nil, http.ErrMissingFile
}
