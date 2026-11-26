package handlers

import (
	"net/http"
	"strings"

	"tracklab/internal/api"
)

// A Garmin client belongs to one login session. Its key, exchanged token
// and responses are never written to the account file or the shared cache.
func (a *Accounts) RegisterGarmin(mux *http.ServeMux) {
	build := func(key string) http.Handler {
		routes := http.NewServeMux()
		NewGarminHandler(api.NewGarminClient(key, nil)).Register(routes)
		return routes
	}
	unconfigured := build("")

	mux.HandleFunc("GET /api/garmin/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		a.mu.Lock()
		_, session, ok := a.session(r)
		a.mu.Unlock()
		if !ok || r.Header.Get("X-TrackLab-Session") != session.binding {
			Error(w, http.StatusUnauthorized, "Connexion requise.")
			return
		}
		if session.garmin == nil {
			unconfigured.ServeHTTP(w, r)
			return
		}
		session.garmin.ServeHTTP(w, r)
	})

	origin := http.NewCrossOriginProtection()
	mux.Handle("PUT /api/garmin/key", origin.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var input struct{ Key string }
		if !readAccountJSON(w, r, 4096, &input) {
			return
		}
		key := strings.TrimSpace(input.Key)
		if key != "" && (len(key) <= len("GCDC_") || len(key) > 1024 ||
			!strings.HasPrefix(key, "GCDC_") || strings.ContainsFunc(key, func(c rune) bool {
			return c < 33 || c > 126
		})) {
			Error(w, http.StatusBadRequest, "Une clé API personnelle Garmin commençant par GCDC_ est attendue.")
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		token, session, ok := a.session(r)
		if !ok || r.Header.Get("X-TrackLab-Session") != session.binding {
			Error(w, http.StatusUnauthorized, "Connexion requise.")
			return
		}
		session.garmin = nil
		if key != "" {
			session.garmin = build(key)
		}
		a.sessions[token] = session
		JSON(w, http.StatusOK, map[string]bool{"configured": key != ""})
	})))
}
