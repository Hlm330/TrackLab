package api

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

const geocodingTTL = 7 * 24 * time.Hour

// SearchCities recherche les villes correspondant au nom fourni.
func (c *Client) SearchCities(name string) (json.RawMessage, error) {
	name = strings.TrimSpace(name)

	if utf8.RuneCountInString(name) < 2 ||
		utf8.RuneCountInString(name) > 100 {
		return nil, errors.New(
			"le nom de la ville doit contenir entre 2 et 100 caractères",
		)
	}

	params := url.Values{
		"name":     {name},
		"count":    {"10"},
		"language": {"fr"},
		"format":   {"json"},
	}

	endpoint := "https://geocoding-api.open-meteo.com/v1/search?" +
		params.Encode()

	return c.GetJSON(
		endpoint,
		nil,
		"geocoding:v1:"+params.Encode(),
		geocodingTTL,
	)
}
