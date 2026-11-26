package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry représente une donnée mise en cache et sa date d'expiration.
type Entry struct {
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

// request représente une récupération de données en cours.
type request struct {
	done chan struct{}
	data json.RawMessage
	err  error
}

// Cache combine un stockage en mémoire et une persistance sur disque.
type Cache struct {
	directory string
	mu        sync.Mutex
	entries   map[string]Entry
	pending   map[string]*request
}

// New initialise le cache et son répertoire de stockage.
func New(directory string) (*Cache, error) {
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("création du cache : %w", err)
	}

	return &Cache{
		directory: directory,
		entries:   make(map[string]Entry),
		pending:   make(map[string]*request),
	}, nil
}

// path transforme une clé en nom de fichier sécurisé.
func (c *Cache) path(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(
		c.directory,
		hex.EncodeToString(hash[:])+".json",
	)
}

// clone protège les données internes contre les modifications externes.
func clone(data json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), data...)
}

// Get récupère une donnée valide depuis la mémoire ou le disque.
func (c *Cache) Get(key string) (json.RawMessage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.getLocked(key)
}

// getLocked doit être appelée avec le verrou déjà acquis.
func (c *Cache) getLocked(key string) (json.RawMessage, bool) {
	now := time.Now()

	if entry, exists := c.entries[key]; exists {
		if now.Before(entry.ExpiresAt) {
			return clone(entry.Data), true
		}

		delete(c.entries, key)
	}

	content, err := os.ReadFile(c.path(key))
	if err != nil {
		return nil, false
	}

	var entry Entry

	if err := json.Unmarshal(content, &entry); err != nil ||
		!json.Valid(entry.Data) ||
		!now.Before(entry.ExpiresAt) {

		_ = os.Remove(c.path(key))
		return nil, false
	}

	c.entries[key] = entry

	return clone(entry.Data), true
}

// Set enregistre une donnée valide avec une durée de vie définie.
func (c *Cache) Set(
	key string,
	data json.RawMessage,
	ttl time.Duration,
) error {
	if key == "" {
		return errors.New("clé de cache vide")
	}

	if ttl <= 0 {
		return errors.New("durée de cache invalide")
	}

	if !json.Valid(data) {
		return errors.New("données JSON invalides")
	}

	now := time.Now()

	entry := Entry{
		Data:      clone(data),
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	content, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encodage du cache : %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.CreateTemp(c.directory, ".cache-*")
	if err != nil {
		return fmt.Errorf("création du fichier temporaire : %w", err)
	}

	defer os.Remove(file.Name())
	defer file.Close()

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("écriture du cache : %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("fermeture du cache : %w", err)
	}

	if err := os.Rename(file.Name(), c.path(key)); err != nil {
		return fmt.Errorf("enregistrement du cache : %w", err)
	}

	c.entries[key] = entry

	return nil
}

// GetOrFetch récupère une donnée en cache ou appelle sa source.
// Les requêtes simultanées portant sur une même clé sont mutualisées.
func (c *Cache) GetOrFetch(
	key string,
	ttl time.Duration,
	fetch func() (json.RawMessage, error),
) (json.RawMessage, error) {
	if key == "" || ttl <= 0 || fetch == nil {
		return nil, errors.New("paramètres de cache invalides")
	}

	c.mu.Lock()

	if data, exists := c.getLocked(key); exists {
		c.mu.Unlock()
		return data, nil
	}

	if current, exists := c.pending[key]; exists {
		c.mu.Unlock()

		<-current.done

		return clone(current.data), current.err
	}

	current := &request{
		done: make(chan struct{}),
	}

	c.pending[key] = current
	c.mu.Unlock()

	data, err := fetch()

	if err == nil {
		if !json.Valid(data) {
			err = errors.New("réponse JSON invalide")
		} else {
			err = c.Set(key, data, ttl)
		}
	}

	c.mu.Lock()

	current.data = clone(data)
	current.err = err

	delete(c.pending, key)
	close(current.done)

	c.mu.Unlock()

	return clone(data), err
}

// Delete supprime une donnée du cache.
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)

	err := os.Remove(c.path(key))

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("suppression du cache : %w", err)
	}

	return nil
}
