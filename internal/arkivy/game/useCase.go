package game

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type gameService struct {
	repo GameRepository
}

func NewService(repo GameRepository) GameService {
	return &gameService{repo: repo}
}

func (s *gameService) CreateGame(name string, amount int) (*Game, error) {
	if name == "" {
		return nil, fmt.Errorf("name es requerido")
	}
	if amount < 0 {
		return nil, fmt.Errorf("amount no puede ser negativo")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("error al generar UUID: %w", err)
	}

	now := time.Now().UTC()
	game := &Game{
		ID:     id.String(),
		Name:   name,
		Amount: amount,
		CAt:    now,
		UAt:    now,
	}

	if err := s.repo.Save(game); err != nil {
		return nil, fmt.Errorf("error al guardar game: %w", err)
	}

	return game, nil
}

func (s *gameService) ListGames() ([]Game, error) {
	games, err := s.repo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("error al listar games: %w", err)
	}
	return games, nil
}
