package test1

import (
	"context"
)

// UserHandler estructura el controlador web para los usuarios
type UserHandler struct {
	uc UserUseCase
}

func NewUserHandler(uc UserUseCase) *UserHandler {
	return &UserHandler{
		uc: uc,
	}
}

// ProcessGoogleUser se llamará desde el main.go para ejecutar la lógica de Clean Architecture
func (h *UserHandler) ProcessGoogleUser(ctx context.Context, userData map[string]interface{}) (*Usuario, error) {
	return h.uc.SignInWithGoogle(ctx, userData)
}
