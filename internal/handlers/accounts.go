package handlers

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const sessionAge = 12 * time.Hour
const accountCookie = "tracklab_session"

type account struct {
	Salt []byte
	Hash []byte
	Data map[string]json.RawMessage
}

type accountSession struct {
	email, binding string
	expires        time.Time
	garmin         http.Handler
}

type Accounts struct {
	mu            sync.Mutex
	path          string
	users         map[string]account
	sessions      map[string]accountSession
	attemptWindow time.Time
	attempts      int
}

func NewAccounts(path string) (*Accounts, error) {
	a := &Accounts{path: path, users: make(map[string]account), sessions: make(map[string]accountSession)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return a, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, &a.users); err != nil {
		return nil, err
	}
	if a.users == nil {
		return nil, errors.New("fichier de comptes invalide")
	}
	return a, nil
}

func (a *Accounts) save() error {
	dir := filepath.Dir(a.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, ".accounts-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = json.NewEncoder(file).Encode(a.users)
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), a.path)
}

func readAccountJSON(w http.ResponseWriter, r *http.Request, limit int64, dst any) bool {
	if strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "application/json" {
		Error(w, http.StatusUnsupportedMediaType, "Requête JSON attendue.")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	if err := decoder.Decode(dst); err != nil || decoder.Decode(new(any)) != io.EOF {
		Error(w, http.StatusBadRequest, "Requête invalide.")
		return false
	}
	return true
}

func (a *Accounts) Register(mux *http.ServeMux) {
	origin := http.NewCrossOriginProtection()
	register := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, origin.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			handler(w, r)
		})))
	}
	register("POST /api/account/register", a.authenticate(true))
	register("POST /api/account/login", a.authenticate(false))
	register("GET /api/account", a.current)
	register("PUT /api/account/data/{key}", a.update)
	register("POST /api/account/logout", a.logout)
}

func (a *Accounts) authenticate(create bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Email, Password string }
		if !readAccountJSON(w, r, 4096, &input) {
			return
		}
		email := strings.ToLower(strings.TrimSpace(input.Email))
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || len(email) > 254 || len(input.Password) > 256 ||
			(create && len([]rune(input.Password)) < 12) {
			Error(w, http.StatusBadRequest, "Adresse e-mail invalide ou mot de passe trop court (12 caractères minimum à l’inscription).")
			return
		}

		a.mu.Lock()
		defer a.mu.Unlock()
		now := time.Now()
		if now.Sub(a.attemptWindow) >= time.Minute {
			a.attemptWindow, a.attempts = now, 0
		}
		if a.attempts >= 20 {
			Error(w, http.StatusTooManyRequests, "Trop de tentatives. Réessayez dans une minute.")
			return
		}
		a.attempts++
		user, exists := a.users[email]
		if !exists {
			user.Salt = make([]byte, 16)
			rand.Read(user.Salt)
		}
		hash, err := pbkdf2.Key(sha256.New, input.Password, user.Salt, 600000, 32)
		if err != nil {
			Error(w, http.StatusInternalServerError, "Connexion indisponible.")
			return
		}
		if create {
			if exists {
				Error(w, http.StatusConflict, "Impossible de créer ce compte.")
				return
			}
			user.Hash, user.Data = hash, make(map[string]json.RawMessage)
			a.users[email] = user
			if err := a.save(); err != nil {
				delete(a.users, email)
				Error(w, http.StatusInternalServerError, "Enregistrement du compte indisponible.")
				return
			}
		} else if !exists || subtle.ConstantTimeCompare(hash, user.Hash) != 1 {
			Error(w, http.StatusUnauthorized, "Adresse e-mail ou mot de passe incorrect.")
			return
		}

		for token, session := range a.sessions {
			if !session.expires.After(now) {
				delete(a.sessions, token)
			}
		}
		if old, err := r.Cookie(accountCookie); err == nil {
			delete(a.sessions, old.Value)
		}
		token, binding := rand.Text(), rand.Text()
		a.sessions[token] = accountSession{email: email, binding: binding, expires: now.Add(sessionAge)}
		setAccountCookie(w, r, token, int(sessionAge.Seconds()))
		JSON(w, http.StatusOK, map[string]any{"email": email, "binding": binding, "data": user.Data})
	}
}

func setAccountCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: accountCookie, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode})
}

// Call with the account lock held.
func (a *Accounts) session(r *http.Request) (string, accountSession, bool) {
	cookie, err := r.Cookie(accountCookie)
	if err != nil {
		return "", accountSession{}, false
	}
	s, ok := a.sessions[cookie.Value]
	return cookie.Value, s, ok && s.expires.After(time.Now())
}

func (a *Accounts) current(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, s, ok := a.session(r)
	if !ok {
		Error(w, http.StatusUnauthorized, "Connexion requise.")
		return
	}
	JSON(w, http.StatusOK, map[string]any{"email": s.email, "binding": s.binding, "data": a.users[s.email].Data})
}

func (a *Accounts) update(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var data json.RawMessage
	if key != "profile" && key != "sessions" && key != "training" {
		Error(w, http.StatusNotFound, "Donnée inconnue.")
		return
	}
	if !readAccountJSON(w, r, 2<<20, &data) {
		return
	}
	if (key == "sessions" && data[0] != '[') || (key != "sessions" && data[0] != '{') {
		Error(w, http.StatusBadRequest, "Données invalides.")
		return
	}
	if key == "sessions" {
		var sessions []json.RawMessage
		if err := json.Unmarshal(data, &sessions); err != nil || len(sessions) > 5000 {
			Error(w, http.StatusBadRequest, "La limite de 5 000 séances est atteinte.")
			return
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, s, ok := a.session(r)
	if !ok || r.Header.Get("X-TrackLab-Session") != s.binding {
		Error(w, http.StatusUnauthorized, "Connexion requise.")
		return
	}
	user := a.users[s.email]
	if user.Data == nil {
		user.Data = make(map[string]json.RawMessage)
		a.users[s.email] = user
	}
	previous, hadPrevious := user.Data[key]
	if hadPrevious && r.Header.Get("If-None-Match") == "*" {
		Error(w, http.StatusPreconditionFailed, "Ces données existent déjà dans le compte.")
		return
	}
	user.Data[key] = data
	if err := a.save(); err != nil {
		if hadPrevious {
			user.Data[key] = previous
		} else {
			delete(user.Data, key)
		}
		Error(w, http.StatusInternalServerError, "Enregistrement indisponible.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *Accounts) logout(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	token, s, ok := a.session(r)
	if ok && r.Header.Get("X-TrackLab-Session") != s.binding {
		Error(w, http.StatusUnauthorized, "La session a changé.")
		return
	}
	delete(a.sessions, token)
	setAccountCookie(w, r, "", -1)
	w.WriteHeader(http.StatusNoContent)
}
