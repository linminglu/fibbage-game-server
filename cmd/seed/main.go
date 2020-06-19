package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/common/log"
	"github.com/zdarovich/fibbage-game-server/internal/db/models"
	"path/filepath"
	"strings"
)

var config Config

type Config struct {
	Normals []Normal `json:"normal"`
}
type Normal struct {
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	Answer             string   `json:"answer"`
	AlternateSpellings []string `json:"alternateSpellings"`
	Suggestions        []string `json:"suggestions"`
}

func init() {
	filepaths, _ := filepath.Glob(filepath.Join("questions_ru.json"))
	if err := configor.Load(&config, filepaths...); err != nil {
		panic(err)
	}

}
func TruncateTables(db *gorm.DB, tables ...interface{}) {
	for _, table := range tables {
		if err := db.DropTableIfExists(table).Error; err != nil {
			panic(err)
		}

		db.AutoMigrate(table)
	}
}

func createQuestions(db *gorm.DB) {
	for _, p := range config.Normals {
		ques := strings.Replace(p.Question, "<BLANK>", "______", -1)
		ques = strings.Replace(ques, "<i>", "", -1)
		ques = strings.Replace(ques, "<i/>", "", -1)
		q := models.Question{
			Category:           p.Category,
			Question:           ques,
			Answer:             p.Answer,
			AlternateSpellings: strings.Join(p.AlternateSpellings, ","),
			Suggestions:        strings.Join(p.Suggestions, ","),
			LangCode:           "ru",
		}
		if err := db.Create(&q).Error; err != nil {
			log.Error(err)
			log.Infof("%+v", q)
		}
	}
}

func main() {
	connStr := fmt.Sprintf(
		"%s:%s@(%s)/fibbage_db?charset=utf8&parseTime=True&loc=Local",
		"newuser",
		"password",
		"localhost",
	)
	db, err := gorm.Open("mysql", connStr)
	if err != nil {
		panic(err)
	}
	//var Tables = []interface{}{
	//	&models.Question{},
	//}
	//TruncateTables(db,Tables...)
	//log.Info("Start create sample data...")

	createQuestions(db)
	log.Info("--> Created questions.")

}
