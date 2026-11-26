package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountIsolationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "accounts.json")
	newMux := func() *http.ServeMux {
		t.Helper()
		accounts, err := NewAccounts(path)
		if err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		accounts.Register(mux)
		accounts.RegisterGarmin(mux)
		return mux
	}
	mux := newMux()
	request := func(method, path, body, binding string, cookie *http.Cookie, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-TrackLab-Session", binding)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, want, w.Body)
		}
		return w
	}
	auth := func(action, email, password string) (*http.Cookie, string) {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"email": email, "password": password})
		w := request("POST", "/api/account/"+action, string(body), "", nil, 200)
		var session struct{ Binding string }
		if err := json.Unmarshal(w.Body.Bytes(), &session); err != nil || session.Binding == "" {
			t.Fatalf("missing session binding: %v", err)
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
			t.Fatal("insecure or missing session cookie")
		}
		return cookies[0], session.Binding
	}
	const password = "synthetic-test-password"
	request("GET", "/api/account", "", "", nil, 401)
	first, firstBinding := auth("register", "first@example.test", password)
	second, secondBinding := auth("register", "second@example.test", password)
	const garminKeyPath = "/api/garmin/key"
	const garminStatusPath = "/api/garmin/status"
	request("GET", garminStatusPath, "", "", nil, 401)
	request("PUT", garminKeyPath, `{"key":"GCDC_FICTIONAL_TEST_ONLY"}`, firstBinding, second, 401)
	request("PUT", garminKeyPath, `{"key":"invalid"}`, firstBinding, first, 400)
	configured := request("PUT", garminKeyPath, `{"key":"GCDC_FICTIONAL_TEST_ONLY"}`, firstBinding, first, 200)
	if !strings.Contains(configured.Body.String(), `"configured":true`) || strings.Contains(configured.Body.String(), "GCDC_") {
		t.Fatal("invalid Garmin configuration response")
	}
	isolated := request("GET", garminStatusPath, "", secondBinding, second, 200)
	if !strings.Contains(isolated.Body.String(), `"configured":false`) {
		t.Fatal("Garmin configuration leaked to another account")
	}
	const dataPath = "/api/account/data/sessions"
	const sessions = `[{"id":"example-session","duration_minutes":35}]`
	request("PUT", dataPath, sessions, firstBinding, second, 401)
	request("PUT", dataPath, sessions, firstBinding, first, 204)
	secondState := request("GET", "/api/account", "", secondBinding, second, 200)
	if strings.Contains(secondState.Body.String(), "example-session") {
		t.Fatal("session leaked to another account")
	}
	request("POST", "/api/account/logout", "", firstBinding, first, 204)
	request("GET", "/api/account", "", "", first, 401)
	mux = newMux()
	request("POST", "/api/account/login", `{"email":"first@example.test","password":"wrong-password"}`, "", nil, 401)
	first, firstBinding = auth("login", "first@example.test", password)
	cleared := request("GET", garminStatusPath, "", firstBinding, first, 200)
	if !strings.Contains(cleared.Body.String(), `"configured":false`) {
		t.Fatal("Garmin configuration persisted after logout")
	}
	state := request("GET", "/api/account", "", firstBinding, first, 200)
	if !strings.Contains(state.Body.String(), "example-session") {
		t.Fatal("session lost after logout and server restart")
	}
}
