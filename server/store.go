package main

import (
	"errors"
	"sync"
)

type Session struct {
	// seed is sufficient to recreate the PRNG.
	seed              int64
	// progress is sufficient to resume using the PRNG to catch up with the client.
	progress          uint32

	// totalMsg is the initial parameter given by the client. It is necessary to determine the stopping condition.
	totalMsg          uint32

	// lastActivityEpoch is the high watermark to implement the state expiration.
	// int64 is used instead of time.Time to be memory efficient; the latter stores pointer to Location.
	lastActivityEpoch int64
}

type store interface {
	Read(clientID string) (Session, error)
	Store(clientID string, sess Session) error

	Delete(clientID string) error
}

type inMemoryStore struct {
	mu      sync.RWMutex
	sessMap map[string]Session
}

var ErrNotFound = errors.New("not found")

func (i *inMemoryStore) Read(clientID string) (Session, error) {
	i.mu.RLock()
	sess, ok := i.sessMap[clientID]
	i.mu.RUnlock()
	if ok {
		return sess, nil
	}
	return Session{}, ErrNotFound
}

func (i *inMemoryStore) Store(clientID string, sess Session) error {
	i.mu.Lock()
	i.sessMap[clientID] = sess
	i.mu.Unlock()
	return nil
}

func (i *inMemoryStore) Delete(clientID string) error {
	i.mu.Lock()
	delete(i.sessMap, clientID)
	i.mu.Unlock()
	return nil
}


var _ store = &inMemoryStore{}
