package api

import (
	"encoding/json"
	"time"
)

const airQualityTTL = time.Hour

// GetAirQuality récupère les indicateurs de qualité de l'air.
func (c *Client) GetAirQuality(
	latitude, longitude float64,
) (json.RawMessage, error) {
	if err := ValidateCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	params := coordinateParams(latitude, longitude)

	params.Set("current",
		"european_aqi,"+
			"pm10,"+
			"pm2_5,"+
			"nitrogen_dioxide,"+
			"ozone",
	)

	params.Set("hourly",
		"european_aqi,"+
			"pm10,"+
			"pm2_5,"+
			"nitrogen_dioxide,"+
			"ozone",
	)

	params.Set("forecast_days", "7")

	endpoint := "https://air-quality-api.open-meteo.com/v1/air-quality?" +
		params.Encode()

	return c.GetJSON(
		endpoint,
		nil,
		"airquality:v1:"+params.Encode(),
		airQualityTTL,
	)
}
