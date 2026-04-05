package test1

import (
	"context"
	"errors"
)

type userUseCaseConfig struct {
	ds UserDataStore
}

// NewUserUseCase crea un nuevo UseCase instanciado
func NewUserUseCase(ds UserDataStore) UserUseCase {
	return &userUseCaseConfig{
		ds: ds,
	}
}

// SignInWithGoogle procesa la info devuelta por Google OAuth
func (uc *userUseCaseConfig) SignInWithGoogle(ctx context.Context, googleData map[string]interface{}) (*Usuario, error) {
	googleID, ok := googleData["id"].(string)
	if !ok {
		return nil, errors.New("no se encontró el ID de Google en los datos proporcionados")
	}

	// 1. Verificar si el usuario ya existe para evitar duplicados
	user, err := uc.ds.FindByGoogleID(ctx, googleID)
	if err != nil {
		return nil, err // Error de base de datos
	}

	// 2. Si el usuario ya existe, simplemente lo retornamos (Flow de Iniciar Sesión)
	if user != nil {
		return user, nil
	}

	// 3. Si no existe, lo creamos (Flow de Registro Automático / Sign Up)
	email, _ := googleData["email"].(string)
	name, _ := googleData["name"].(string)
	picture, _ := googleData["picture"].(string)

	newUser := &Usuario{
		GoogleID: googleID,
		Email:    email,
		Name:     name,
		Picture:  picture,
	}

	err = uc.ds.CreateUser(ctx, newUser)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}
