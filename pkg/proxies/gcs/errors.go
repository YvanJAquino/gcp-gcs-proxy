package gcs

import (
	"encoding/json"
	"log"
	"net/http"
)

type Error struct {
	Description string `json:"error"`
}

func (e Error) Error() string {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

var (
	ErrBadRequest          = &Error{"400 Bad Request: please review your URL"}
	ErrBucketDoesNotExist  = &Error{"400 Bad Request: bucket does not exist"}
	ErrObjectDoesNotExist  = &Error{"400 Bad Request: object does not exist"}
	ErrInternalServerError = &Error{"500 Internal Server Error: something went wrong"}
)

func HTTPErrBadRequest(w http.ResponseWriter) {
	http.Error(w, ErrBadRequest.Error(), http.StatusBadRequest)
}

func HTTPErrBucketDoesNotExist(w http.ResponseWriter) {
	http.Error(w, ErrBucketDoesNotExist.Error(), http.StatusBadRequest)
}

func HTTPErrObjectDoesNotExist(w http.ResponseWriter) {
	http.Error(w, ErrObjectDoesNotExist.Error(), http.StatusBadRequest)
}

func HTTPErrInternalServerError(w http.ResponseWriter, err error) {
	http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
	go log.Println(err)
}
