package memory

import (
	"cows/domain"
	"errors"
	"sync"
)

type Store struct {
	mu            sync.RWMutex
	matches       map[string]*domain.Match
	activeByUser  map[string]string
	processedInts map[string]struct{}
}

func New() *Store {
	return &Store{
		matches:       map[string]*domain.Match{},
		activeByUser:  map[string]string{},
		processedInts: map[string]struct{}{},
	}
}

func cloneMatch(m *domain.Match) *domain.Match {
	if m == nil {
		return nil
	}
	cpy := *m
	cpy.History = append([]domain.TurnResult(nil), m.History...)
	return &cpy
}

func (s *Store) CreateMatch(match *domain.Match) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.matches[match.ID]; ok {
		return errors.New("match already exists")
	}
	s.matches[match.ID] = cloneMatch(match)
	return nil
}

func (s *Store) UpdateMatch(match *domain.Match) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.matches[match.ID]; !ok {
		return errors.New("match not found")
	}
	s.matches[match.ID] = cloneMatch(match)
	return nil
}

func (s *Store) GetMatch(id string) (*domain.Match, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.matches[id]
	if !ok {
		return nil, errors.New("match not found")
	}
	return cloneMatch(m), nil
}

func (s *Store) GetActiveMatchByPlayer(playerID string) (*domain.Match, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.activeByUser[playerID]
	if !ok {
		return nil, errors.New("no active match")
	}
	m, ok := s.matches[id]
	if !ok {
		return nil, errors.New("match missing")
	}
	return cloneMatch(m), nil
}

func (s *Store) SetActiveMatch(playerID, matchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, busy := s.activeByUser[playerID]; busy {
		return errors.New("player already in active match")
	}
	s.activeByUser[playerID] = matchID
	return nil
}

func (s *Store) ClearActiveMatch(playerID, matchID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.activeByUser[playerID]; ok && id == matchID {
		delete(s.activeByUser, playerID)
	}
	return nil
}

func (s *Store) WasInteractionProcessed(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.processedInts[id]
	return ok, nil
}

func (s *Store) MarkInteractionProcessed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processedInts[id] = struct{}{}
	return nil
}
