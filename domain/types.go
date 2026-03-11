package domain

import (
	"errors"
	"fmt"
)

type GameState string

const (
	StateWaitingAccept          GameState = "WAITING_ACCEPT"
	StateWaitingSecretP1        GameState = "WAITING_SECRET_P1"
	StateWaitingSecretP2        GameState = "WAITING_SECRET_P2"
	StateWaitingSecretConfirmP1 GameState = "WAITING_SECRET_CONFIRM_P1"
	StateWaitingSecretConfirmP2 GameState = "WAITING_SECRET_CONFIRM_P2"
	StateInProgressP1Turn       GameState = "IN_PROGRESS_P1_TURN"
	StateInProgressP2Turn       GameState = "IN_PROGRESS_P2_TURN"
	StateFinished               GameState = "FINISHED"
	StateCancelled              GameState = "CANCELLED"
	StateExpired                GameState = "EXPIRED"
)

type TurnResult struct {
	PlayerID string
	Guess    string
	Bulls    int
	Cows     int
}

type Match struct {
	ID              string
	Player1ID       string
	Player2ID       string
	State           GameState
	Secret1         string
	Secret2         string
	PendingSecret1  string
	PendingSecret2  string
	PendingGuess    string
	PendingGuessBy  string
	History         []TurnResult
	WinnerID        string
	Version         int64
	ChallengeMsgRef string
	CreatedBy       string
	LastInteraction string
	TimeoutToken    string
}

func ValidateSecretCode(code string) error {
	if len(code) != 4 {
		return errors.New("code must be 4 digits")
	}
	seen := make(map[rune]bool, 4)
	for _, r := range code {
		if r < '0' || r > '9' {
			return errors.New("code must contain digits only")
		}
		if seen[r] {
			return errors.New("digits must be unique")
		}
		seen[r] = true
	}
	return nil
}

func EvaluateBullsCows(secret, guess string) (int, int, error) {
	if err := ValidateSecretCode(secret); err != nil {
		return 0, 0, fmt.Errorf("invalid secret: %w", err)
	}
	if err := ValidateSecretCode(guess); err != nil {
		return 0, 0, fmt.Errorf("invalid guess: %w", err)
	}
	bulls, cows := 0, 0
	for i := range 4 {
		if guess[i] == secret[i] {
			bulls++
			continue
		}
		for j := range 4 {
			if i != j && guess[i] == secret[j] {
				cows++
				break
			}
		}
	}
	return bulls, cows, nil
}
