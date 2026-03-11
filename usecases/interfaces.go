package usecases

import (
	"cows/domain"
	"time"
)

type Store interface {
	CreateMatch(match *domain.Match) error
	UpdateMatch(match *domain.Match) error
	GetMatch(id string) (*domain.Match, error)
	GetActiveMatchByPlayer(playerID string) (*domain.Match, error)
	SetActiveMatch(playerID, matchID string) error
	ClearActiveMatch(playerID, matchID string) error
	WasInteractionProcessed(id string) (bool, error)
	MarkInteractionProcessed(id string) error
}

type Scheduler interface {
	Schedule(key string, after time.Duration, fn func())
	Cancel(key string)
}

type Locker interface {
	WithKey(key string, fn func() error) error
}
