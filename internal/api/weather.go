package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"strconv"
	"time"
)

const weatherTTL = 30 * time.Minute

// ValidateCoordinates vérifie la validité des coordonnées géographiques.
func ValidateCoordinates(latitude, longitude float64) error {
	if math.IsNaN(latitude) || math.IsNaN(longitude) ||
		math.IsInf(latitude, 0) || math.IsInf(longitude, 0) {
		return errors.New("coordonnées géographiques invalides")
	}

	if latitude < -90 || latitude > 90 ||
		longitude < -180 || longitude > 180 {
		return errors.New("coordonnées géographiques hors limites")
	}

	return nil
}

// coordinateParams normalise les coordonnées utilisées par les API.
func coordinateParams(latitude, longitude float64) url.Values {
	return url.Values{
		"latitude": {
			strconv.FormatFloat(latitude, 'f', 4, 64),
		},
		"longitude": {
			strconv.FormatFloat(longitude, 'f', 4, 64),
		},
		"timezone": {"auto"},
	}
}

// GetWeather récupère les conditions actuelles et les prévisions.
func (c *Client) GetWeather(
	latitude, longitude float64,
) (json.RawMessage, error) {
	if err := ValidateCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	params := coordinateParams(latitude, longitude)

	params.Set("current",
		"temperature_2m,"+
			"relative_humidity_2m,"+
			"apparent_temperature,"+
			"precipitation,"+
			"weather_code,"+
			"wind_speed_10m,"+
			"wind_direction_10m,"+
			"wind_gusts_10m",
	)

	params.Set("hourly",
		"temperature_2m,"+
			"relative_humidity_2m,"+
			"apparent_temperature,"+
			"precipitation_probability,"+
			"precipitation,"+
			"weather_code,"+
			"wind_speed_10m,"+
			"wind_direction_10m,"+
			"wind_gusts_10m,"+
			"uv_index",
	)

	params.Set("forecast_days", "7")
	params.Set("temperature_unit", "celsius")
	params.Set("wind_speed_unit", "kmh")
	params.Set("precipitation_unit", "mm")

	endpoint := "https://api.open-meteo.com/v1/forecast?" +
		params.Encode()

	return c.GetJSON(
		endpoint,
		nil,
		"weather:v1:"+params.Encode(),
		weatherTTL,
	)
}
