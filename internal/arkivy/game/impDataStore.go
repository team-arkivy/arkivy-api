package game

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type mongoRepository struct {
	collection *mongo.Collection
}

func NewRepository(db *mongo.Database) GameRepository {
	return &mongoRepository{
		collection: db.Collection("game"),
	}
}

func (r *mongoRepository) Save(game *Game) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, game)
	if err != nil {
		return fmt.Errorf("error al insertar game: %w", err)
	}
	return nil
}

func (r *mongoRepository) FindAll() ([]Game, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("error al consultar games: %w", err)
	}
	defer cursor.Close(ctx)

	var games []Game
	if err := cursor.All(ctx, &games); err != nil {
		return nil, fmt.Errorf("error al decodificar games: %w", err)
	}
	return games, nil
}
