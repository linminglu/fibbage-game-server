package models

import (
	"github.com/jinzhu/gorm"
)

type User struct {
	gorm.Model
	Name         string
	Score        int
	ConnectionID string
	RoomID       int
	Room         Room
	QuestionID   int
	Question     Question
}
