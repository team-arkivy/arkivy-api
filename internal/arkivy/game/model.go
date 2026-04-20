package game

import (
	"time"
)

type Game struct {
	ID     string    `bson:"_id" json:"id"`
	Name   string    `bson:"name" json:"name"`
	Amount int       `bson:"amount" json:"amount"`
	CAt    time.Time `bson:"c_at" json:"cAt"`
	UAt    time.Time `bson:"u_at" json:"uAt"`
}
