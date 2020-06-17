package models

import "github.com/jinzhu/gorm"

type Score struct {
	gorm.Model
	UserID int
	User   User
}
