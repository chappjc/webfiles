// Copyright (c) 2018 Jonathan Chappelow
// See LICENSE for details.

package server

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chappjc/webfiles/middleware"
	"github.com/chappjc/webfiles/response"

	"github.com/OneOfOne/xxhash"
	"github.com/go-chi/chi"
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

// Server manages cookies/auth, and implements the http handlers
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

// File is the handler for file downloads, requiring the "{fileid}" URL path
// parameter (e.g. /file/{fileid}).
func (s *Server) File(w http.ResponseWriter, r *http.Request) {
	// Extract the file's unique id from the path
	fileID := chi.URLParam(r, "fileid")

	// Verify authentication
	if !middleware.RequestCtxAuthed(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check token
	token := middleware.RequestCtxToken(r)
	if !s.FileAuthCheck(fileID, token) {
		http.Error(w, "unauthorized for file "+fileID, http.StatusUnauthorized)
		return
	}

	// Locate file in storage by it's UID
	fullFile, statusCode, err := s.UIDToFilePath(fileID, false)
	if err != nil {
		log.Errorln(err)
		http.Error(w, err.Error(), statusCode)
		return
	}

	// Send the file
	if err = response.SendFile(w, fullFile); err != nil {
		log.Errorln(err)
		http.Error(w, err.Error(), statusCode)
		return
	}
}

func (s *Server) FileList(w http.ResponseWriter, r *http.Request) {
	// check token
	// serve file list
}

func (s *Server) FileAuthCheck(fileID, token string) bool {
	// check with user-file DB
	return true
}

// Token returns the user/session's current JWT.
func (s *Server) Token(w http.ResponseWriter, r *http.Request) {
	userJWT := middleware.RequestCtxToken(r)
	if userJWT == "" {
		http.Error(w, "JWT not available", http.StatusInternalServerError)
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
		http.Error(w, "JWT not available", http.StatusInternalServerError)
		return
	}
	fmt.Println("session ID for upload: ", session.ID)

	switch r.Method {
	// POST
	case http.MethodPost:
		// Get media type from Content-Type header
		contentType := r.Header.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, "file upload Content-Type request must be multipart/form-data",
				http.StatusUnsupportedMediaType)
			return
		}

		// Ensure it is a multipart/... media type
		if !strings.HasPrefix(mediaType, "multipart/") {
			http.Error(w, "invalid Content-Type "+mediaType, http.StatusBadRequest)
		}

		// Process the multipart.File upload
		mpFile, fileHeader, err := s.formFile(uploadPostParam, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer mpFile.Close()

		// Compute UID of file. Use a non-cryptographic hash function for speed.
		hasher := xxhash.New64()
		numBytes, err := io.Copy(hasher, mpFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// UID is a 16 character hex string (8 bytes of data)
		UID := fmt.Sprintf("%016x", hasher.Sum64())
		log.Infof("Hashed %d bytes. UID: %s", numBytes, UID)

		_, err = mpFile.Seek(0, io.SeekStart)
		if err != nil {
			log.Errorln(err)
		}

		// Copy file to storage folder, creating the folder first.
		fullPath, _ := filepath.Abs(filepath.Join(s.FilesPath, UID))
		if err = os.MkdirAll(fullPath, 0755); err != nil {
			log.Errorln(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Combine path and file name, then sanitize it.
		fullFile := filepath.Join(fullPath, fileHeader.Filename)
		fullFile = filepath.Clean(fullFile) // eliminates ".."
		// Do not allow user to write outside of storage path.
		if !strings.HasPrefix(fullFile, fullPath) {
			http.Error(w, os.ErrPermission.Error(), http.StatusBadRequest)
			return
		}

		// Copy upload to storage folder
		fid, err := os.OpenFile(fullFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Errorln(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer fid.Close()

		numBytesStored, err := io.Copy(fid, mpFile)
		if err != nil {
			log.Errorln(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if numBytesStored != numBytes {
			log.Errorf("File %d not stored completely. %d B hashed, %d B copied",
				fullFile, numBytes, numBytesStored)
		}

		// Store the original file name in a text file "NAME".
		err = ioutil.WriteFile(filepath.Join(fullPath, "NAME"),
			[]byte(fileHeader.Filename), 0644)
		if err != nil {
			log.Errorln(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Write success response to user
		resp := &response.UploadResponse{
			Upload: response.Upload{
				UID:      UID,
				FileName: fileHeader.Filename,
				Size:     numBytes,
			},
			Token: userJWT,
		}
		response.WriteJSON(w, resp, "    ")

	default:
		http.Error(w, "must be POST", http.StatusMethodNotAllowed)
		return
	}
}

// UIDToFilePath looks up the file name for the file with unique identifier UID,
// and returns the absolute path to the files, a http status code, and an error.
func (s *Server) UIDToFilePath(UID string, mkdir bool) (string, int, error) {
	// Get full path to file in storage location
	fullPath, _ := filepath.Abs(filepath.Join(s.FilesPath, UID))
	if mkdir {
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return "", http.StatusInternalServerError, err
		}
		// TODO if existed, return notfound error
	}

	// Get original name of file
	fName, err := ioutil.ReadFile(filepath.Join(fullPath, "NAME"))
	if err != nil || len(fName) == 0 {
		log.Errorf("NAME file in %d unreadable: %v", fullPath, err)
		return "", http.StatusInternalServerError, err
	}

	fullFile := filepath.Join(fullPath, string(fName))
	fullFile = filepath.Clean(fullFile)
	if !strings.HasPrefix(fullFile, fullPath) {
		return "", http.StatusBadRequest, os.ErrPermission
	}
	return fullFile, http.StatusOK, nil
}

var multipartByReader = &multipart.Form{
	Value: make(map[string][]string),
	File:  make(map[string][]*multipart.FileHeader),
}

// formFile gets the file for the given key (e.g. "fileupload") from the
// request's parsed multipart form, calling ParseMultipartForm first if
// necessary. On success, a non-nil multipart.File and multipart.FileHeader are
// returned, but the error must be checked to ensure it was opened successfully.
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
