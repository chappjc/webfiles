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

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"

	"github.com/OneOfOne/xxhash"
	"github.com/go-chi/jwtauth"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// UseLog sets an external logger for use by this package.
func UseLog(_log *logrus.Logger) {
	log = _log
}

const (
	defaultFilesPath = "uploads"
	uploadPostParam  = "fileupload"
)

// Server manages cookies/auth, and provides the http handlers
type Server struct {
	CookieStore *sessions.FilesystemStore
	AuthToken   *jwtauth.JWTAuth
	SigningKey  string
	MaxFileSize int64
	FilesPath   string
}

// NewServer creates a new Server for the given signing secret, cookie storage
// file system path, and uploaded file size limit.
func NewServer(secret, cookieStorePath string, maxFileSize int64) *Server {
	shaSum := sha256.Sum256([]byte(secret))
	server := &Server{
		SigningKey:  secret,
		CookieStore: sessions.NewFilesystemStore(cookieStorePath, shaSum[:]),
		AuthToken:   jwtauth.New("HS256", []byte(secret), nil),
		MaxFileSize: maxFileSize,
		FilesPath:   defaultFilesPath,
	}

	opts := server.CookieStore.Options
	opts.Path = "/"
	opts.HttpOnly = true
	opts.Secure = false // for HTTPS-only, set true

	return server
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "webfiles is running!")
}

func (s *Server) File(w http.ResponseWriter, r *http.Request) {
	// get file id
	// check token
	// serve file
	response.WritePlainText(w, "file")
}

func (s *Server) FileList(w http.ResponseWriter, r *http.Request) {
	// check token
	// serve file list
}

// Token returns the user/session's current JWT.
func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	userJWT := middleware.RequestCtxToken(r)
	if userJWT == "" {
		response.Error(w, http.StatusInternalServerError, "JWT not available")
		return
	}
	response.WritePlainText(w, userJWT)
}

// UploadFile is the upload handler for POST requests with the file data stored
// in the body with Content-Type multipart/form-data.
func (s *Server) UploadFile(w http.ResponseWriter, r *http.Request) {
	session := middleware.RequestCtxJWTSession(r)
	userJWT := middleware.RequestCtxToken(r)
	if session == nil || userJWT == "" {
		response.Error(w, http.StatusInternalServerError, "JWT not available")
		return
	}
	fmt.Println("session ID for upload: ", session.ID)

	switch r.Method {
	// POST
	case http.MethodPost:
		// V
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			response.Error(w, http.StatusUnsupportedMediaType,
				"file upload Content-Type request must be multipart/form-data")
			return
		}
		log.Debugln(params)

		if !strings.HasPrefix(mediaType, "multipart/") {
			response.Error(w, http.StatusBadRequest, "invalid Content-Type "+mediaType)
		}

		// Process the multipart.File upload
		mpFile, fileHeader, err := s.formFile(uploadPostParam, r)
		if err != nil {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		defer mpFile.Close()

		// Compute UID of file
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

		// Copy file to storage folder
		fullPath, _ := filepath.Abs(filepath.Join(s.FilesPath, UID))
		if err = os.MkdirAll(fullPath, 0755); err != nil {
			log.Errorln(err)
			response.Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		fullFile := filepath.Join(fullPath, fileHeader.Filename)
		fullFile = filepath.Clean(fullFile)
		if !strings.HasPrefix(fullFile, fullPath) {
			response.Error(w, http.StatusBadRequest, os.ErrPermission.Error())
			return
		}

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

		// Write success response to user
		resp := &response.Upload{
			UID:      UID,
			FileName: fileHeader.Filename,
			Size:     numBytes,
		}
		response.WriteJSON(w, resp, "    ")

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
