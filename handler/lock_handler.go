package handler

import (
	"encoding/json"
	"net/http"

	"github.com/dimitrovvlado/pglock/db"
)

//LockRequest it the struct representing a request for a lock
type LockRequest struct {
	ProfileID string `json:"profileId"`
	DeviceID  string `json:"deviceId"`
}

//LockReponse is a struct representing the response of a lock request
type LockReponse struct {
	ProfileID string `json:"profileId"`
	DeviceID  string `json:"deviceId"`
	TTL       int    `json:"ttl"`
}

//LockHandle is a HTTP handler for the /v1/lock endpoint
func LockHandle(w http.ResponseWriter, r *http.Request) {
	switch method := r.Method; method {
	case "POST":
		postLockHandle(w, r)
	case "DELETE":
		deleteLockHandle(w, r)
	}
}

func postLockHandle(w http.ResponseWriter, r *http.Request) {
	var l LockRequest
	err := json.NewDecoder(r.Body).Decode(&l)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, []byte("Can't parse Lock request: "+err.Error()))
		return
	}
	locked, err := db.AttemptLock(l.ProfileID, l.DeviceID)
	if err != nil {
		writeResponse(w, http.StatusServiceUnavailable, []byte(err.Error()))
		return
	}
	if locked {
		body, _ := json.Marshal(&LockReponse{TTL: int(db.DefaultTTL)})
		writeResponse(w, http.StatusOK, body)
		return
	}
	writeResponse(w, http.StatusBadRequest, []byte{})
}

func deleteLockHandle(w http.ResponseWriter, r *http.Request) {
	var l LockRequest
	err := json.NewDecoder(r.Body).Decode(&l)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, []byte("Can't parse Lock request: "+err.Error()))
		return
	}
	rows, err := db.ReleaseLock(l.ProfileID)
	if err != nil {
		writeResponse(w, http.StatusServiceUnavailable, []byte(err.Error()))
		return
	}
	if rows > 0 {
		writeResponse(w, http.StatusOK, []byte{})
		return
	}
	writeResponse(w, http.StatusNotFound, []byte{})
}

func writeResponse(w http.ResponseWriter, status int, body []byte) {
	w.WriteHeader(status)
	w.Header().Set("Content-Length", string(len(body)))
	w.Write(body)
}
