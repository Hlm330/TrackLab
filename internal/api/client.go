package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"tracklab/internal/cache"
)

const maxResponseSize int64 = 10 * 1024 * 1024

// Client centralise les communications avec les API externes.
type Client struct {
	httpClient *http.Client
	cache      *cache.Cache
}

// NewClient initialise un client HTTP sécurisé.
func NewClient(storage *cache.Cache) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,

			// Les redirections sont refusées pour éviter
			// la transmission involontaire des identifiants.
			CheckRedirect: func(
				req *http.Request,
				via []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
		cache: storage,
	}
}

// GetJSON récupère une réponse JSON avec mise en cache facultative.
func (c *Client) GetJSON(
	endpoint string,
	headers map[string]string,
	cacheKey string,
	ttl time.Duration,
) (json.RawMessage, error) {
	parsedURL, err := url.Parse(endpoint)

	if err != nil ||
		parsedURL.Scheme != "https" ||
		parsedURL.Hostname() == "" ||
		parsedURL.User != nil {
		return nil, errors.New("adresse API invalide")
	}

	if !isAllowedHost(parsedURL.Hostname()) {
		return nil, errors.New("domaine API non autorisé")
	}

	if parsedURL.Port() != "" && parsedURL.Port() != "443" {
		return nil, errors.New("port API non autorisé")
	}

	fetch := func() (json.RawMessage, error) {
		return c.fetchJSON(endpoint, headers)
	}

	if c.cache == nil || cacheKey == "" || ttl <= 0 {
		return fetch()
	}

	return c.cache.GetOrFetch(cacheKey, ttl, fetch)
}

// isAllowedHost limite les communications aux API autorisées.
func isAllowedHost(host string) bool {
	switch strings.ToLower(host) {
	case "api.open-meteo.com",
		"air-quality-api.open-meteo.com",
		"geocoding-api.open-meteo.com",
		"connect.garmin.com":
		return true

	default:
		return false
	}
}

// fetchJSON exécute une requête HTTP et valide sa réponse.
func (c *Client) fetchJSON(
	endpoint string,
	headers map[string]string,
) (json.RawMessage, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		endpoint,
		nil,
	)

	if err != nil {
		return nil, errors.New("création de la requête impossible")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "TrackLab/1.0")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.New("API externe inaccessible")
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, errors.New("authentification API invalide")

	case http.StatusForbidden:
		return nil, errors.New("accès API refusé")

	case http.StatusTooManyRequests:
		return nil, errors.New("limite de requêtes API atteinte")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"réponse API inattendue : HTTP %d",
			resp.StatusCode,
		)
	}

	content, err := io.ReadAll(
		io.LimitReader(resp.Body, maxResponseSize+1),
	)

	if err != nil {
		return nil, errors.New("lecture de la réponse API impossible")
	}

	if int64(len(content)) > maxResponseSize {
		return nil, errors.New("réponse API trop volumineuse")
	}

	if !json.Valid(content) {
		return nil, errors.New("réponse JSON invalide")
	}

	return json.RawMessage(content), nil
}
