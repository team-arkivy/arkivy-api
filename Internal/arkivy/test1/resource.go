package test1

import "go.mongodb.org/mongo-driver/v2/mongo"

// Resources contiene las dependencias instanciadas necesarias por el sistema de usuarios
type Resources struct {
	Handler *UserHandler
	UseCase UserUseCase
}

// InitUserModule actúa como un contendor de inyección de dependencias
// Enlaza el framework web al UseCase y la Base de Datos
func InitUserModule(db *mongo.Database) *Resources {
	ds := NewUserDataStore(db)
	uc := NewUserUseCase(ds)
	handler := NewUserHandler(uc)

	return &Resources{
		Handler: handler,
		UseCase: uc,
	}
}
