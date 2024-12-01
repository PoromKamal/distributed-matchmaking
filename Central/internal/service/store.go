package serviceapi

import (
	"fmt"
	"sync"
)

// Store is an interface to define generic storage behavior.
type Store interface {
	Create(ip string) error
	Read() ([]string, error)
	Delete(ip string) error
	Patch(ip string) (string, error)
}

// InMemoryStore is a thread-safe implementation of the Store interface.
type InMemoryStore struct {
	data map[string]bool
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
			data: make(map[string]bool),
		}
	})
	return instance
}

// Create adds an IP-to-username mapping to the store.
func (s *InMemoryStore) Create(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[ip]; exists {
		return fmt.Errorf("IP %s is already registered", ip)
	}

	s.data[ip] = true
	return nil
}

// Read retrieves the ips of all chat servers which are up
func (s *InMemoryStore) Read() ([]string, error) {
	s.mu.RLock() // Lock for read-only access
	defer s.mu.RUnlock()

	var activeIPs []string
	for ip, isActive := range s.data {
		if isActive {
			activeIPs = append(activeIPs, ip)
		}
	}

	return activeIPs, nil
}

// Soft delete, just set the up status to false
func (s *InMemoryStore) Delete(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range s.data {
		if k == ip {
			s.data[k] = false
			return nil
		}
	}
	return fmt.Errorf("IP %s not found", ip)
}

/* Patch sets the status of the ip to true */
func (s *InMemoryStore) Patch(ip string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for k := range s.data {
		if k == ip {
			s.data[k] = true
			return ip, nil
		}
	}
	return "", fmt.Errorf("IP %s not found", ip)
}
