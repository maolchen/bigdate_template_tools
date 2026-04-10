package server

import (
	"strings"
	"sync"
)

// getAISessionLock returns a stable mutex for one session id to serialize mutations.
func (s *Server) getAISessionLock(sessionID string) *sync.Mutex {
	key := strings.TrimSpace(sessionID)
	if key == "" {
		return &sync.Mutex{}
	}
	if loaded, ok := s.aiSessionLocks.Load(key); ok {
		if lock, valid := loaded.(*sync.Mutex); valid {
			return lock
		}
	}
	lock := &sync.Mutex{}
	actual, _ := s.aiSessionLocks.LoadOrStore(key, lock)
	casted, _ := actual.(*sync.Mutex)
	if casted != nil {
		return casted
	}
	return lock
}

// deleteAISessionLock drops the lock object for a removed session.
func (s *Server) deleteAISessionLock(sessionID string) {
	key := strings.TrimSpace(sessionID)
	if key == "" {
		return
	}
	s.aiSessionLocks.Delete(key)
}

