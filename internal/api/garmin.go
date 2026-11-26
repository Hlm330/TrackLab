package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"tracklab/internal/cache"
)

const (
	garminBaseURL     = "https://personal.connectapi.garmin.com"
	garminTokenPath   = "/personalapitoken-service/personal-api/personal-api-key/token"
	garminMaxResponse = 10 * 1024 * 1024
)

type GarminClient struct {
	apiKey     string
	httpClient *http.Client
	storage    *cache.Cache

	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

func NewGarminClient(
	apiKey string,
	storage *cache.Cache,
) *GarminClient {
	return &GarminClient{
		apiKey:  strings.TrimSpace(apiKey),
		storage: storage,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(
				req *http.Request,
				via []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (g *GarminClient) Configured() bool {
	return g.apiKey != ""
}

func (g *GarminClient) validateKey() error {
	if !g.Configured() {
		return errors.New("Garmin non configuré")
	}

	if !strings.HasPrefix(g.apiKey, "GCDC_") {
		return errors.New("format de clé Garmin invalide")
	}

	for _, character := range g.apiKey {
		if character < 33 || character > 126 {
			return errors.New(
				"caractères interdits dans la clé Garmin",
			)
		}
	}

	return nil
}

// Lit la date d'expiration du JWT.
// Cette lecture sert uniquement au renouvellement du token.
func garminTokenExpiration(
	token string,
) (time.Time, error) {
	parts := strings.Split(token, ".")

	if len(parts) != 3 {
		return time.Time{}, errors.New("token Garmin invalide")
	}

	payload, err := base64.RawURLEncoding.DecodeString(
		parts[1],
	)

	if err != nil {
		return time.Time{}, errors.New("JWT Garmin invalide")
	}

	var claims struct {
		Expiration int64 `json:"exp"`
	}

	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, errors.New("expiration Garmin invalide")
	}

	expiration := time.Unix(claims.Expiration, 0)

	if !expiration.After(time.Now()) {
		return time.Time{}, errors.New("token Garmin expiré")
	}

	return expiration, nil
}

// Authentification et renouvellement automatique.
// Le token reste exclusivement en mémoire.
func (g *GarminClient) accessToken() (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.validateKey(); err != nil {
		return "", err
	}

	if g.token != "" &&
		time.Now().Add(5*time.Minute).Before(g.expiresAt) {
		return g.token, nil
	}

	request, err := http.NewRequest(
		http.MethodPost,
		garminBaseURL+garminTokenPath,
		nil,
	)

	if err != nil {
		return "", errors.New("requête Garmin invalide")
	}

	request.Header.Set("Authorization", "ApiKey "+g.apiKey)
	request.Header.Set("Accept", "*/*")
	request.Header.Set("User-Agent", "TrackLab/1.0")

	response, err := g.httpClient.Do(request)

	if err != nil {
		return "", errors.New("Garmin inaccessible")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"authentification Garmin refusée : HTTP %d",
			response.StatusCode,
		)
	}

	body, err := io.ReadAll(
		io.LimitReader(response.Body, 32769),
	)

	if err != nil || len(body) > 32768 {
		return "", errors.New("réponse Garmin invalide")
	}

	token := strings.TrimSpace(string(body))

	expiration, err := garminTokenExpiration(token)

	if err != nil {
		return "", err
	}

	g.token = token
	g.expiresAt = expiration

	return g.token, nil
}

func (g *GarminClient) invalidateToken(token string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.token == token {
		g.token = ""
		g.expiresAt = time.Time{}
	}
}

func (g *GarminClient) fetch(
	path string,
	params url.Values,
) (json.RawMessage, error) {
	endpoint := garminBaseURL + path

	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	for attempt := 0; attempt < 2; attempt++ {
		token, err := g.accessToken()

		if err != nil {
			return nil, err
		}

		request, err := http.NewRequest(
			http.MethodGet,
			endpoint,
			nil,
		)

		if err != nil {
			return nil, errors.New("requête Garmin invalide")
		}

		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Accept", "*/*")
		request.Header.Set("User-Agent", "TrackLab/1.0")

		response, err := g.httpClient.Do(request)

		if err != nil {
			return nil, errors.New("Garmin inaccessible")
		}

		status := response.StatusCode

		body, readErr := io.ReadAll(
			io.LimitReader(
				response.Body,
				garminMaxResponse+1,
			),
		)

		response.Body.Close()

		if status == http.StatusUnauthorized &&
			attempt == 0 {
			g.invalidateToken(token)
			continue
		}

		if status == http.StatusNoContent {
			return json.RawMessage("null"), nil
		}

		if status != http.StatusOK {
			return nil, fmt.Errorf(
				"Garmin : HTTP %d",
				status,
			)
		}

		if readErr != nil ||
			len(body) > garminMaxResponse ||
			!json.Valid(body) {
			return nil, errors.New("données Garmin invalides")
		}

		return json.RawMessage(body), nil
	}

	return nil, errors.New("autorisation Garmin refusée")
}

func (g *GarminClient) get(
	path string,
	params url.Values,
	ttl time.Duration,
) (json.RawMessage, error) {
	if err := g.validateKey(); err != nil {
		return nil, err
	}

	fetch := func() (json.RawMessage, error) {
		return g.fetch(path, params)
	}

	if g.storage == nil {
		return fetch()
	}

	key := "garmin:" + path + "?" + params.Encode()

	// Isole les entrées si la clé personnelle est remplacée.
	key = garminCacheKey(g.apiKey, key)

	return g.storage.GetOrFetch(key, ttl, fetch)
}

func (g *GarminClient) GetActivities(
	limit int,
) (json.RawMessage, error) {
	if limit < 1 || limit > 100 {
		return nil, errors.New(
			"nombre d'activités invalide",
		)
	}

	return g.get(
		"/activitylist-service/activities/search/activities",
		url.Values{
			"start": {"0"},
			"limit": {strconv.Itoa(limit)},
		},
		15*time.Minute,
	)
}

func validateGarminDate(date string) error {
	_, err := time.Parse("2006-01-02", date)

	if err != nil {
		return errors.New("date invalide")
	}

	return nil
}

func (g *GarminClient) GetSleep(
	date string,
) (json.RawMessage, error) {
	if err := validateGarminDate(date); err != nil {
		return nil, err
	}

	return g.get(
		"/sleep-service/sleep/dailySleep",
		url.Values{"date": {date}},
		30*time.Minute,
	)
}

func (g *GarminClient) GetBodyBattery(
	date string,
) (json.RawMessage, error) {
	if err := validateGarminDate(date); err != nil {
		return nil, err
	}

	return g.get(
		"/wellness-service/wellness/bodyBattery/reports/daily",
		url.Values{
			"startDate": {date},
			"endDate":   {date},
		},
		10*time.Minute,
	)
}

func (g *GarminClient) GetHeartRate(
	date string,
) (json.RawMessage, error) {
	if err := validateGarminDate(date); err != nil {
		return nil, err
	}

	return g.get(
		"/wellness-service/wellness/dailyHeartRate",
		url.Values{"date": {date}},
		15*time.Minute,
	)
}

func (g *GarminClient) GetHRV(
	date string,
) (json.RawMessage, error) {
	if err := validateGarminDate(date); err != nil {
		return nil, err
	}

	return g.get(
		"/hrv-service/hrv/"+date,
		nil,
		30*time.Minute,
	)
}

func (g *GarminClient) GetDailySummary(
	date string,
) (json.RawMessage, error) {
	if err := validateGarminDate(date); err != nil {
		return nil, err
	}

	return g.get(
		"/usersummary-service/v2/dailysummaries",
		url.Values{
			"startDate": {date},
			"endDate":   {date},
		},
		15*time.Minute,
	)
}
