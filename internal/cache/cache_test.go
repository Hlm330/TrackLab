package cache

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()

	c, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestCacheSetAndGet(t *testing.T) {
	c := newTestCache(t)

	data := json.RawMessage(`{"temperature":18}`)

	if err := c.Set("weather", data, time.Minute); err != nil {
		t.Fatal(err)
	}

	result, exists := c.Get("weather")

	if !exists || string(result) != string(data) {
		t.Fatal("les données récupérées sont incorrectes")
	}
}

func TestCachePersistence(t *testing.T) {
	directory := t.TempDir()

	first, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}

	data := json.RawMessage(`{"wind":12}`)

	if err := first.Set("weather", data, time.Minute); err != nil {
		t.Fatal(err)
	}

	second, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}

	result, exists := second.Get("weather")

	if !exists || string(result) != string(data) {
		t.Fatal("la persistance du cache a échoué")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := newTestCache(t)

	err := c.Set(
		"expired",
		json.RawMessage(`{"value":1}`),
		time.Nanosecond,
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, exists := c.Get("expired"); exists {
		t.Fatal("une donnée expirée ne doit pas être retournée")
	}
}

func TestCacheInvalidJSON(t *testing.T) {
	c := newTestCache(t)

	err := c.Set(
		"invalid",
		json.RawMessage(`invalid JSON`),
		time.Minute,
	)

	if err == nil {
		t.Fatal("un JSON invalide doit être refusé")
	}
}

func TestCacheCorruptedFile(t *testing.T) {
	c := newTestCache(t)

	err := os.WriteFile(
		c.path("corrupted"),
		[]byte("fichier corrompu"),
		0600,
	)

	if err != nil {
		t.Fatal(err)
	}

	if _, exists := c.Get("corrupted"); exists {
		t.Fatal("un fichier corrompu doit être ignoré")
	}
}

func TestCacheConcurrentRequests(t *testing.T) {
	c := newTestCache(t)

	var calls atomic.Int32
	var wg sync.WaitGroup

	fetch := func() (json.RawMessage, error) {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond)

		return json.RawMessage(`{"value":42}`), nil
	}

	for i := 0; i < 20; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			data, err := c.GetOrFetch(
				"shared",
				time.Minute,
				fetch,
			)

			if err != nil || string(data) != `{"value":42}` {
				t.Error("récupération concurrente incorrecte")
			}
		}()
	}

	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("1 appel attendu, %d obtenus", calls.Load())
	}
}

func TestCacheDoesNotStoreErrors(t *testing.T) {
	c := newTestCache(t)

	_, err := c.GetOrFetch(
		"failed",
		time.Minute,
		func() (json.RawMessage, error) {
			return nil, errors.New("API indisponible")
		},
	)

	if err == nil {
		t.Fatal("l'erreur doit être retournée")
	}

	if _, exists := c.Get("failed"); exists {
		t.Fatal("une réponse en erreur ne doit pas être stockée")
	}
}

func TestCacheDelete(t *testing.T) {
	c := newTestCache(t)

	if err := c.Set(
		"delete",
		json.RawMessage(`{"value":1}`),
		time.Minute,
	); err != nil {
		t.Fatal(err)
	}

	if err := c.Delete("delete"); err != nil {
		t.Fatal(err)
	}

	if _, exists := c.Get("delete"); exists {
		t.Fatal("la donnée aurait dû être supprimée")
	}
}

func TestCacheDataIsolation(t *testing.T) {
	c := newTestCache(t)

	data := json.RawMessage(`{"value":1}`)

	if err := c.Set("isolated", data, time.Minute); err != nil {
		t.Fatal(err)
	}

	data[0] = 'X'

	result, exists := c.Get("isolated")
	if !exists || string(result) != `{"value":1}` {
		t.Fatal("les données internes ont été modifiées")
	}

	result[0] = 'X'

	result, exists = c.Get("isolated")
	if !exists || string(result) != `{"value":1}` {
		t.Fatal("les données retournées ne sont pas isolées")
	}
}
