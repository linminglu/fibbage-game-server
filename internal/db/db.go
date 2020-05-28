package db

import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

var DB *gorm.DB

func init() {
	// Migrate the schema
	var err error
	DB, err = gorm.Open("mysql", "newuser:password@/fibbage_db?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic("failed to connect database")
	}

}
