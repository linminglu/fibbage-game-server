package models

import "github.com/jinzhu/gorm"

type Question struct {
	gorm.Model
	ModeType           ModeType
	Category           string `json:"category"`
	Question           string `json:"question"`
	Answer             string `json:"answer"`
	AlternateSpellings string `json:"alternateSpellings"`
	Suggestions        string `json:"suggestions"`
}

type QuestionAnswerEntity struct {
	gorm.Model
	QuestionID uint
	AnswerType AnswerType
	Value      string
	UserID     uint
	User       User
}

type AnswerType int

const (
	OPTION AnswerType = iota
	TEXT
)
