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
	UpdateDelayList(username string, delays map[string]float32) error
	GetDelayList(username string) (map[string]float32, error)
	InsertChatInstance(roomId string, chatServer string, users []string) (string, error)
	RemoveChatInstance(roomId string) (string, error)
	RemoveChatInstancesForServer(server string) ([]string, error)
	RemoveChatInstancesForUser(user string) (string, error)
}

type ChatInstance struct {
	chatServer string
	users      []string
	roomId     string
	active     bool
}

// InMemoryStore is a thread-safe implementation of the Store interface.
type InMemoryStore struct {
	data          map[string]string
	delayLists    map[string]map[string]float32 // username --> server --> delay
	chatInstances []ChatInstance
	mu            sync.RWMutex
}

var (
	instance *InMemoryStore
	once     sync.Once
)

// GetInMemoryStore returns the singleton instance of InMemoryStore.
func GetInMemoryStore() *InMemoryStore {
	once.Do(func() {
		instance = &InMemoryStore{
			data:          make(map[string]string),
			delayLists:    make(map[string]map[string]float32),
			chatInstances: []ChatInstance{},
		}
	})
	return instance
}

// Create adds an IP-to-username mapping to the store.
func (s *InMemoryStore) Create(ip, username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *InMemoryStore) UpdateDelayList(username string, delays map[string]float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.delayLists[username] = delays
	return nil
}

func (s *InMemoryStore) GetDelayList(username string) (map[string]float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	delays, exists := s.delayLists[username]
	if !exists {
		return nil, fmt.Errorf("delays for username %s not found", username)
	}
	return delays, nil
}

func (s *InMemoryStore) InsertChatInstance(roomId string, chatServer string, users []string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	newInstance := ChatInstance{roomId: roomId, chatServer: chatServer, users: users, active: true}
	s.chatInstances = append(s.chatInstances, newInstance)
	return roomId, nil
}
func (s *InMemoryStore) RemoveChatInstance(roomId string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, instance := range s.chatInstances {
		if instance.roomId == roomId {
			s.chatInstances = append(s.chatInstances[:i], s.chatInstances[i+1:]...)
			return roomId, nil
		}
	}
	return "", fmt.Errorf("chat instance with roomId %s not found", roomId)
}

func (s *InMemoryStore) RemoveChatInstancesForServer(server string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var removedInstances []string
	for i, instance := range s.chatInstances {
		if instance.chatServer == server {
			removedInstances = append(removedInstances, instance.roomId)
			s.chatInstances = append(s.chatInstances[:i], s.chatInstances[i+1:]...)
		}
	}
	return removedInstances, nil
}

func (s *InMemoryStore) RemoveChatInstancesForUser(user string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i, instance := range s.chatInstances {
		for j, u := range instance.users {
			if u == user {
				instance.users = append(instance.users[:j], instance.users[j+1:]...)
				if len(instance.users) == 0 {
					s.chatInstances = append(s.chatInstances[:i], s.chatInstances[i+1:]...)
					return instance.roomId, nil
				}
			}
		}
	}
	return "", fmt.Errorf("chat instance with user %s not found", user)
}
