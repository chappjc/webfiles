package response

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

type Upload struct {
	UID      string `json:"uid"`
	FileName string `json:"file_name"`
	Size     int64  `json:"file_size"`
}

func UseLog(_log *logrus.Logger) {
	log = _log
}

func WriteJSONHandlerFunc(obj interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, obj, "    ")
	}
}

func WriteJSON(w http.ResponseWriter, obj interface{}, indent string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", indent)
	if err := encoder.Encode(obj); err != nil {
		log.Infof("JSON encode error: %v", err)
	}
}

func Error(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, message)
}
