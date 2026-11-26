package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config contient les paramètres nécessaires au serveur.
type Config struct {
	Host        string
	Port        string
	GarminToken string
}

// Load charge la configuration locale et les variables d'environnement.
func Load() (Config, error) {
	if err := loadEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Host:        getEnv("HOST", "127.0.0.1"),
		Port:        getEnv("PORT", "8080"),
		GarminToken: os.Getenv("GARMIN_API_TOKEN"),
	}

	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, errors.New("port HTTP invalide")
	}

	// TrackLab est réservé à une utilisation locale.
	if cfg.Host != "127.0.0.1" && cfg.Host != "::1" {
		return Config{}, errors.New(
			"le serveur doit utiliser une adresse locale",
		)
	}

	return cfg, nil
}

// getEnv retourne une variable ou sa valeur par défaut.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// loadEnv charge un fichier de configuration simple.
// Les variables déjà présentes dans l'environnement restent prioritaires.
func loadEnv(path string) error {
	file, err := os.Open(path)

	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("lecture de la configuration : %w", err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++

		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")

		if !found || strings.TrimSpace(key) == "" {
			return fmt.Errorf(
				"configuration invalide à la ligne %d",
				lineNumber,
			)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf(
					"variable invalide à la ligne %d",
					lineNumber,
				)
			}
		}
	}

	return scanner.Err()
}
