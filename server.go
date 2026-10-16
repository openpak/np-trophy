package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Trophies is what the HTTP layer needs from storage.
type Trophies interface {
	Merge(ctx context.Context, account, communicationID string, unlocks []Unlock) error
	Games(ctx context.Context, account string) ([]Game, error)
}

const (
	maxBody     = 64 << 10
	maxTrophies = 1024
)

// Handler serves the private API np-account calls:
//
//	GET  /healthz                          no auth
//	GET  /v1/users/{core-uuid}/trophies    every stored unlock
//	POST /v1/users/{core-uuid}/trophies    {"communication_id", "trophies": [{id, earned_at, grade}]}
//
// Both trophy routes answer {"games": [...]}, the POST after merging.
func Handler(store Trophies, key string, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		reply(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/v1/users/{account}/trophies", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Values("Authorization")
		if len(auth) > 1 {
			reply(w, http.StatusBadRequest, errorBody("duplicate authorization"))
			return
		}
		if len(auth) == 0 || subtle.ConstantTimeCompare([]byte(auth[0]), []byte("Bearer "+key)) != 1 {
			reply(w, http.StatusUnauthorized, errorBody("unauthorized"))
			return
		}
		account := r.PathValue("account")
		if !validAccount(account) {
			reply(w, http.StatusBadRequest, errorBody("invalid core account UUID"))
			return
		}
		switch r.Method {
		case http.MethodGet:
		case http.MethodPost:
			var in struct {
				CommunicationID string   `json:"communication_id"`
				Trophies        []Unlock `json:"trophies"`
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
			if err := dec.Decode(&in); err != nil {
				var tooBig *http.MaxBytesError
				if errors.As(err, &tooBig) {
					reply(w, http.StatusRequestEntityTooLarge, errorBody("body too large"))
				} else {
					reply(w, http.StatusBadRequest, errorBody("invalid JSON"))
				}
				return
			}
			if !validCommunicationID(in.CommunicationID) || in.Trophies == nil || len(in.Trophies) > maxTrophies {
				reply(w, http.StatusBadRequest, errorBody("invalid trophy set"))
				return
			}
			for _, u := range in.Trophies {
				if u.ID < 0 || u.ID >= maxTrophies || u.EarnedAt < 0 || !validGrade(u.Grade) {
					reply(w, http.StatusBadRequest, errorBody("invalid trophy"))
					return
				}
			}
			if err := store.Merge(r.Context(), account, in.CommunicationID, in.Trophies); err != nil {
				log.Error("trophy merge", "err", err)
				reply(w, http.StatusInternalServerError, errorBody("storage unavailable"))
				return
			}
		default:
			reply(w, http.StatusMethodNotAllowed, errorBody("method not allowed"))
			return
		}
		games, err := store.Games(r.Context(), account)
		if err != nil {
			log.Error("trophy read", "err", err)
			reply(w, http.StatusInternalServerError, errorBody("storage unavailable"))
			return
		}
		reply(w, http.StatusOK, map[string]any{"games": games})
	})
	return mux
}

func errorBody(msg string) map[string]string { return map[string]string{"error": msg} }

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// validGrade accepts no grade or bronze, silver, gold, platinum.
func validGrade(g string) bool { return g == "" || g == "B" || g == "S" || g == "G" || g == "P" }

// validAccount accepts only the core's canonical lower-case UUID form.
func validAccount(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f'):
			return false
		}
	}
	return true
}

// validCommunicationID accepts NPWR01234_00: nine upper-case letters or digits, '_', two digits.
func validCommunicationID(s string) bool {
	if len(s) != 12 || s[9] != '_' {
		return false
	}
	for i := 0; i < 12; i++ {
		c := s[i]
		switch {
		case i == 9:
		case i > 9:
			if c < '0' || c > '9' {
				return false
			}
		case !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'):
			return false
		}
	}
	return true
}
