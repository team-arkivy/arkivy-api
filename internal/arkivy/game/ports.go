package game

// Puerto de salida — hacia MongoDB
type GameRepository interface {
	Save(game *Game) error
	FindAll() ([]Game, error)
}

// Puerto de entrada — casos de uso expuestos
type GameService interface {
	CreateGame(name string, amount int) (*Game, error)
	ListGames() ([]Game, error)
}
