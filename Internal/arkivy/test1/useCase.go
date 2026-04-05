package test1

import "context"

// UserUseCase contiene la lógica de negocio para la gestión de usuarios
type UserUseCase interface {
	SignInWithGoogle(ctx context.Context, googleData map[string]interface{}) (*Usuario, error)
}
