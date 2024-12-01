package clientapi

import (
	"fmt"
	"sync"
)

// Store is an interface to define generic storage behavior.
type Store interface {
	Create(ip, username string) error
	Read(ip string) (string, error)
	Delete(ip string) error
	ReadByUsername(username string) (string, error)
}

// InMemoryStore is a thread-safe implementation of the Store interface.
type InMemoryStore struct {
	data map[string]string
	mu   sync.RWMutex
}

var (
	instance *InMemoryStore
	once     sync.Once
)

// GetInMemoryStore returns the singleton instance of InMemoryStore.
func GetInMemoryStore() *InMemoryStore {
	once.Do(func() {
		instance = &InMemoryStore{
			data: make(map[string]string),
		}
	})
	return instance
}

// Create adds an IP-to-username mapping to the store.
func (s *InMemoryStore) Create(ip, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[ip]; exists {
		return fmt.Errorf("IP %s is already registered", ip)
	}

	s.data[ip] = username
	return nil
}

// Read retrieves the username for a given IP.
func (s *InMemoryStore) Read(ip string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	username, exists := s.data[ip]
	if !exists {
		return "", fmt.Errorf("IP %s not found", ip)
	}

	return username, nil
}

// ReadByUsername retrieves the IP for a given username.
func (s *InMemoryStore) ReadByUsername(username string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for ip, u := range s.data {
		if u == username {
			return ip, nil
		}
	}

	return "", fmt.Errorf("Username %s not found", username)
}

// Delete removes an IP-to-username mapping from the store.
func (s *InMemoryStore) Delete(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[ip]; !exists {
		return fmt.Errorf("IP %s not found", ip)
	}

	delete(s.data, ip)
	return nil
}
