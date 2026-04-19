package test1

import "context"

// UserDataStore es la interfaz para manejar datos de usuario en la BD
type UserDataStore interface {
	CreateUser(ctx context.Context, user *Usuario) error
	FindByGoogleID(ctx context.Context, googleID string) (*Usuario, error)
}
