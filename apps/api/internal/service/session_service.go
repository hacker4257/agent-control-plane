package service

import (
	"context"

	"agent-control-plane/apps/api/internal/repo"
)

type SessionStore interface {
	ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error)
	GetSessionByID(ctx context.Context, sessionID string) (repo.SessionRecord, error)
	ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error)
}

type SessionService struct {
	store SessionStore
}

func NewSessionService(store SessionStore) *SessionService {
	return &SessionService{store: store}
}

func (s *SessionService) ListSessions(ctx context.Context, limit, offset int) ([]repo.SessionRecord, error) {
	if s == nil || s.store == nil {
		return []repo.SessionRecord{}, nil
	}
	return s.store.ListSessions(ctx, limit, offset)
}

func (s *SessionService) GetSession(ctx context.Context, sessionID string) (repo.SessionRecord, error) {
	if s == nil || s.store == nil {
		return repo.SessionRecord{SessionID: sessionID}, nil
	}
	return s.store.GetSessionByID(ctx, sessionID)
}

func (s *SessionService) ListSessionTimeline(ctx context.Context, sessionID string, limit, offset int) ([]repo.EventRecord, error) {
	if s == nil || s.store == nil {
		return []repo.EventRecord{}, nil
	}
	return s.store.ListSessionEvents(ctx, sessionID, limit, offset)
}
