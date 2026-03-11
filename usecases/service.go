package usecases

import (
	"cows/domain"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

type Durations struct {
	ChallengeTimeout time.Duration
	SecretTimeout    time.Duration
	ConfirmTimeout   time.Duration
	TurnTimeout      time.Duration
}

type Service struct {
	store  Store
	sched  Scheduler
	lock   Locker
	timing Durations
}

func NewService(store Store, sched Scheduler, lock Locker, timing Durations) *Service {
	return &Service{store: store, sched: sched, lock: lock, timing: timing}
}

func genID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Service) idempotent(interactionID string) (bool, error) {
	if interactionID == "" {
		return false, nil
	}
	ok, err := s.store.WasInteractionProcessed(interactionID)
	if err != nil || ok {
		return ok, err
	}
	return false, s.store.MarkInteractionProcessed(interactionID)
}

func (s *Service) CreateChallenge(interactionID, challenger, opponent string) (*domain.Match, error) {
	if challenger == opponent {
		return nil, errors.New("you cannot duel yourself")
	}
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	match := &domain.Match{ID: genID(), Player1ID: challenger, Player2ID: opponent, State: domain.StateWaitingAccept, CreatedBy: challenger, Version: 1}
	if err := s.store.SetActiveMatch(challenger, match.ID); err != nil {
		return nil, err
	}
	if err := s.store.SetActiveMatch(opponent, match.ID); err != nil {
		_ = s.store.ClearActiveMatch(challenger, match.ID)
		return nil, err
	}
	if err := s.store.CreateMatch(match); err != nil {
		_ = s.store.ClearActiveMatch(challenger, match.ID)
		_ = s.store.ClearActiveMatch(opponent, match.ID)
		return nil, err
	}
	s.scheduleChallengeExpiry(match.ID)
	return match, nil
}

func (s *Service) AcceptChallenge(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if m.State != domain.StateWaitingAccept {
			return errors.New("challenge is not pending")
		}
		if m.Player2ID != actor {
			return errors.New("only challenged player can accept")
		}
		m.State = domain.StateWaitingSecretP1
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.sched.Cancel("challenge:" + m.ID)
		s.scheduleSecretTimeout(m.ID)
		out = m
		return nil
	})
	return out, err
}

func (s *Service) DeclineChallenge(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if m.State != domain.StateWaitingAccept {
			return errors.New("challenge is not pending")
		}
		if actor != m.Player1ID && actor != m.Player2ID {
			return errors.New("not a participant")
		}
		m.State = domain.StateCancelled
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.finish(m)
		out = m
		return nil
	})
	return out, err
}

func (s *Service) SubmitSecret(interactionID, matchID, actor, secret string) (*domain.Match, error) {
	if err := domain.ValidateSecretCode(secret); err != nil {
		return nil, err
	}
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		switch actor {
		case m.Player1ID:
			if m.Secret1 != "" {
				return errors.New("secret already confirmed")
			}
			m.PendingSecret1 = secret
			m.State = domain.StateWaitingSecretConfirmP1
		case m.Player2ID:
			if m.Secret2 != "" {
				return errors.New("secret already confirmed")
			}
			m.PendingSecret2 = secret
			m.State = domain.StateWaitingSecretConfirmP2
		default:
			return errors.New("not a participant")
		}
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.scheduleSecretAutoConfirm(m.ID, actor)
		out = m
		return nil
	})
	return out, err
}

func (s *Service) ConfirmSecret(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if err := s.confirmSecretNoIdempotent(m, actor); err != nil {
			return err
		}
		out = m
		return nil
	})
	return out, err
}

func (s *Service) confirmSecretNoIdempotent(m *domain.Match, actor string) error {
	switch actor {
	case m.Player1ID:
		if m.PendingSecret1 == "" {
			return errors.New("no pending secret")
		}
		m.Secret1 = m.PendingSecret1
		m.PendingSecret1 = ""
	case m.Player2ID:
		if m.PendingSecret2 == "" {
			return errors.New("no pending secret")
		}
		m.Secret2 = m.PendingSecret2
		m.PendingSecret2 = ""
	default:
		return errors.New("not a participant")
	}
	if m.Secret1 != "" && m.Secret2 != "" {
		m.State = domain.StateInProgressP1Turn
		s.scheduleTurnTimeout(m.ID)
	} else if m.Secret1 == "" {
		m.State = domain.StateWaitingSecretP1
	} else {
		m.State = domain.StateWaitingSecretP2
	}
	m.Version++
	return s.store.UpdateMatch(m)
}

func (s *Service) EditSecret(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if actor == m.Player1ID {
			m.PendingSecret1 = ""
			m.State = domain.StateWaitingSecretP1
		} else if actor == m.Player2ID {
			m.PendingSecret2 = ""
			m.State = domain.StateWaitingSecretP2
		} else {
			return errors.New("not a participant")
		}
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		out = m
		return nil
	})
	return out, err
}

func (s *Service) SubmitGuess(interactionID, matchID, actor, guess string) (*domain.Match, error) {
	if err := domain.ValidateSecretCode(guess); err != nil {
		return nil, err
	}
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if (m.State == domain.StateInProgressP1Turn && actor != m.Player1ID) || (m.State == domain.StateInProgressP2Turn && actor != m.Player2ID) {
			return errors.New("not your turn")
		}
		m.PendingGuess = guess
		m.PendingGuessBy = actor
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.scheduleGuessAutoConfirm(m.ID, actor)
		out = m
		return nil
	})
	return out, err
}

func (s *Service) ConfirmGuess(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if m.PendingGuessBy != actor || m.PendingGuess == "" {
			return errors.New("no pending guess")
		}
		opponentSecret := m.Secret2
		next := domain.StateInProgressP2Turn
		if actor == m.Player2ID {
			opponentSecret = m.Secret1
			next = domain.StateInProgressP1Turn
		}
		bulls, cows, err := domain.EvaluateBullsCows(opponentSecret, m.PendingGuess)
		if err != nil {
			return err
		}
		m.History = append(m.History, domain.TurnResult{PlayerID: actor, Guess: m.PendingGuess, Bulls: bulls, Cows: cows})
		m.PendingGuess = ""
		m.PendingGuessBy = ""
		if bulls == 4 {
			m.State = domain.StateFinished
			m.WinnerID = actor
			s.finish(m)
		} else {
			m.State = next
			s.scheduleTurnTimeout(m.ID)
		}
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		out = m
		return nil
	})
	return out, err
}

func (s *Service) CancelMatch(interactionID, matchID, actor string) (*domain.Match, error) {
	if done, err := s.idempotent(interactionID); err != nil || done {
		return nil, err
	}
	var out *domain.Match
	err := s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		if actor != m.Player1ID && actor != m.Player2ID {
			return errors.New("not a participant")
		}
		m.State = domain.StateCancelled
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.finish(m)
		out = m
		return nil
	})
	return out, err
}

func (s *Service) CreateRematch(interactionID, matchID, actor string) (*domain.Match, error) {
	prev, err := s.store.GetMatch(matchID)
	if err != nil {
		return nil, err
	}
	if prev.State != domain.StateFinished && prev.State != domain.StateCancelled && prev.State != domain.StateExpired {
		return nil, errors.New("rematch allowed only after finish")
	}
	if actor != prev.Player1ID && actor != prev.Player2ID {
		return nil, errors.New("not a participant")
	}
	return s.CreateChallenge(interactionID, prev.Player1ID, prev.Player2ID)
}

func (s *Service) scheduleChallengeExpiry(matchID string) {
	s.sched.Schedule("challenge:"+matchID, s.timing.ChallengeTimeout, func() {
		_ = s.expireMatch(matchID, domain.StateWaitingAccept)
	})
}

func (s *Service) scheduleSecretTimeout(matchID string) {
	s.sched.Schedule("secret:"+matchID, s.timing.SecretTimeout, func() {
		_ = s.expireMatch(matchID, domain.StateWaitingSecretP1, domain.StateWaitingSecretP2, domain.StateWaitingSecretConfirmP1, domain.StateWaitingSecretConfirmP2)
	})
}

func (s *Service) scheduleTurnTimeout(matchID string) {
	s.sched.Schedule("turn:"+matchID, s.timing.TurnTimeout, func() {
		_ = s.expireMatch(matchID, domain.StateInProgressP1Turn, domain.StateInProgressP2Turn)
	})
}

func (s *Service) scheduleSecretAutoConfirm(matchID, actor string) {
	s.sched.Schedule(fmt.Sprintf("secconfirm:%s:%s", matchID, actor), s.timing.ConfirmTimeout, func() {
		_, _ = s.ConfirmSecret("", matchID, actor)
	})
}

func (s *Service) scheduleGuessAutoConfirm(matchID, actor string) {
	s.sched.Schedule(fmt.Sprintf("guessconfirm:%s:%s", matchID, actor), s.timing.ConfirmTimeout, func() {
		_, _ = s.ConfirmGuess("", matchID, actor)
	})
}

func (s *Service) expireMatch(matchID string, allowed ...domain.GameState) error {
	return s.lock.WithKey(matchID, func() error {
		m, err := s.store.GetMatch(matchID)
		if err != nil {
			return err
		}
		ok := false
		for _, st := range allowed {
			if m.State == st {
				ok = true
				break
			}
		}
		if !ok {
			return nil
		}
		m.State = domain.StateExpired
		m.Version++
		if err := s.store.UpdateMatch(m); err != nil {
			return err
		}
		s.finish(m)
		return nil
	})
}

func (s *Service) finish(m *domain.Match) {
	s.sched.Cancel("challenge:" + m.ID)
	s.sched.Cancel("secret:" + m.ID)
	s.sched.Cancel("turn:" + m.ID)
	_ = s.store.ClearActiveMatch(m.Player1ID, m.ID)
	_ = s.store.ClearActiveMatch(m.Player2ID, m.ID)
}
