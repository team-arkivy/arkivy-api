package game

import (
	"time"
)

type Game struct {
	ID     string    `bson:"_id"  json:"id"     example:"018f1a2b-3c4d-7e5f-8a9b-0c1d2e3f4a5b"`
	Name   string    `bson:"name" json:"name"   example:"Chess"`
	Amount int       `bson:"amount" json:"amount" example:"10"`
	CAt    time.Time `bson:"c_at" json:"cAt"`
	UAt    time.Time `bson:"u_at" json:"uAt"`
}

type CreateGameRequest struct {
	Name   string `json:"name"   binding:"required" example:"Chess"`
	Amount int    `json:"amount" binding:"min=0"    example:"10"`
}
