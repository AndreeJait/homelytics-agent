package outbound

import (
	"context"
	"sync"

	"github.com/AndreeJait/go-utility/v2/statusw"
	"github.com/AndreeJait/homelytics-agent/domain/entity"
	portOutbound "github.com/AndreeJait/homelytics-agent/port/outbound"
)

type memorySessionStore struct {
	mu       sync.RWMutex
	session  *entity.AuthSession
	key      *entity.TSNetAuthKey
	hostname string
}

// NewMemorySessionStore creates an in-memory session store.
func NewMemorySessionStore() portOutbound.SessionStore {
	return &memorySessionStore{}
}

func (s *memorySessionStore) SetSession(_ context.Context, session *entity.AuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = session
	return nil
}

func (s *memorySessionStore) GetSession(_ context.Context) (*entity.AuthSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.session == nil {
		return nil, statusw.NotFound.WithCustomMessage("no active session")
	}
	return s.session, nil
}

func (s *memorySessionStore) SetTSNetAuthKey(_ context.Context, key *entity.TSNetAuthKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.key = key
	return nil
}

func (s *memorySessionStore) GetTSNetAuthKey(_ context.Context) (*entity.TSNetAuthKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.key == nil {
		return nil, statusw.NotFound.WithCustomMessage("no tsnet auth key")
	}
	return s.key, nil
}

func (s *memorySessionStore) SetHostname(_ context.Context, hostname string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostname = hostname
	return nil
}

func (s *memorySessionStore) GetHostname(_ context.Context) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.hostname == "" {
		return "", statusw.NotFound.WithCustomMessage("no hostname set")
	}
	return s.hostname, nil
}
