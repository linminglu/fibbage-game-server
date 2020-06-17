package models

import "github.com/jinzhu/gorm"

type Room struct {
	gorm.Model
	Uuid      string
	StateType StateType
	ModeType  ModeType
}
