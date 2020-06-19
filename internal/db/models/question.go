package models

import "github.com/jinzhu/gorm"

type Question struct {
	gorm.Model
	Category           string
	Question           string
	Answer             string
	AlternateSpellings string
	Suggestions        string
	LangCode           string
}

type QuestionTranslation struct {
	gorm.Model
	Code string
}
