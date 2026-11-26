package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tracklab/internal/api"
	"tracklab/internal/cache"
	"tracklab/internal/config"
	"tracklab/internal/handlers"
)

const (
	applicationName    = "TrackLab"
	applicationVersion = "1.0.0"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatal("Configuration invalide : ", err)
	}

	// Cache des données environnementales publiques.
	publicCache, err := cache.New("data/cache")

	if err != nil {
		log.Fatal("Initialisation du cache public impossible : ", err)
	}

	environmentClient := api.NewClient(publicCache)

	// Cache indépendant des données sportives personnelles.
	personalCache, err := cache.New("data/personal/garmin")

	if err != nil {
		log.Fatal("Initialisation du cache personnel impossible : ", err)
	}

	garminClient := api.NewGarminClient(
		cfg.GarminToken,
		personalCache,
	)

	mux := http.NewServeMux()

	// Routes environnementales.
	environmentHandler := handlers.NewEnvironmentHandler(
		environmentClient,
	)

	environmentHandler.Register(mux)

	// Routes sportives personnelles.
	garminHandler := handlers.NewGarminHandler(
		garminClient,
	)

	garminHandler.Register(mux)

	// Routes des moteurs d'analyse sportive.
	analysisHandler := handlers.NewAnalysisHandler()
	analysisHandler.Register(mux)

	// Route de vérification du serveur.
	mux.HandleFunc(
		"GET /api/health",
		func(w http.ResponseWriter, r *http.Request) {
			handlers.JSON(w, http.StatusOK, map[string]any{
				"application":       applicationName,
				"version":           applicationVersion,
				"status":            "ok",
				"garmin_configured": garminClient.Configured(),
			})
		},
	)

	// Seuls les fichiers publics du frontend sont accessibles.
	mux.Handle(
		"GET /css/",
		publicFiles(),
	)

	mux.Handle(
		"GET /js/",
		publicFiles(),
	)

	mux.Handle(
		"GET /assets/",
		publicFiles(),
	)

	mux.HandleFunc(
		"GET /{$}",
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "web/index.html")
		},
	)

	address := net.JoinHostPort(cfg.Host, cfg.Port)

	server := &http.Server{
		Addr: address,
		Handler: securityHeaders(
			validateHost(address, mux),
		),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	go func() {
		log.Printf(
			"%s disponible sur http://%s",
			applicationName,
			address,
		)

		log.Printf(
			"Garmin configuré : %t",
			garminClient.Configured(),
		)

		err := server.ListenAndServe()

		if err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Erreur du serveur : ", err)
		}
	}()

	<-ctx.Done()

	log.Println("Arrêt de TrackLab...")

	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		log.Printf("Erreur pendant l'arrêt : %v", err)
	}

	log.Println("Serveur arrêté.")
}

// Protège le serveur local contre le DNS rebinding.
func validateHost(
	address string,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Host != address {
				handlers.Error(
					w,
					http.StatusForbidden,
					"Hôte non autorisé.",
				)
				return
			}

			next.ServeHTTP(w, r)
		},
	)
}

// Sert uniquement les fichiers publics, sans listing de dossiers.
func publicFiles() http.Handler {
	root := http.Dir("web")
	fileServer := http.FileServer(root)

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			file, err := root.Open(r.URL.Path)

			if err != nil {
				http.NotFound(w, r)
				return
			}

			info, err := file.Stat()
			file.Close()

			if err != nil || !info.Mode().IsRegular() {
				http.NotFound(w, r)
				return
			}

			fileServer.ServeHTTP(w, r)
		},
	)
}

// Applique les protections HTTP à toutes les réponses.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(
				"Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self'; "+
					"style-src 'self'; "+
					"img-src 'self' data:; "+
					"connect-src 'self'; "+
					"object-src 'none'; "+
					"base-uri 'none'; "+
					"frame-ancestors 'none'; "+
					"form-action 'self'",
			)

			w.Header().Set(
				"X-Content-Type-Options",
				"nosniff",
			)

			w.Header().Set(
				"X-Frame-Options",
				"DENY",
			)

			w.Header().Set(
				"Referrer-Policy",
				"no-referrer",
			)

			w.Header().Set(
				"Permissions-Policy",
				"camera=(), microphone=(), geolocation=(self)",
			)

			w.Header().Set(
				"Cross-Origin-Resource-Policy",
				"same-origin",
			)

			next.ServeHTTP(w, r)
		},
	)
}
