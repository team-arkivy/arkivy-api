package test1

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Usuario struct {
	ID       bson.ObjectID `json:"id" bson:"_id,omitempty"`
	GoogleID string        `json:"google_id" bson:"google_id"`
	Email    string        `json:"email" bson:"email"`
	Name     string        `json:"name" bson:"name"`
	Picture  string        `json:"picture" bson:"picture,omitempty"`
}
