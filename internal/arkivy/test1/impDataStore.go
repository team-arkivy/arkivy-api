package test1

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type userDataStoreConfig struct {
	collection *mongo.Collection
}

// NewUserDataStore crea un data store usando la colección "usuarios"
func NewUserDataStore(db *mongo.Database) UserDataStore {
	return &userDataStoreConfig{
		collection: db.Collection("usuarios"),
	}
}

// CreateUser inserta un documento de usuario nuevo
func (ds *userDataStoreConfig) CreateUser(ctx context.Context, user *Usuario) error {
	result, err := ds.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	
	// Para mongo-driver/v2 mapeamos la ID insertada al modelo struct (dependiendo de la ID autogenerada)
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		user.ID = oid
	}
	return nil
}

// FindByGoogleID busca si un usuario ya existe en base a su ID de Google
func (ds *userDataStoreConfig) FindByGoogleID(ctx context.Context, googleID string) (*Usuario, error) {
	var user Usuario
	err := ds.collection.FindOne(ctx, bson.M{"google_id": googleID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Retorna nulo si no encontró al usuario (significa que es registro nuevo)
		}
		return nil, err
	}
	return &user, nil
}
