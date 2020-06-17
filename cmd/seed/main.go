package main

import (
	"fmt"
	"github.com/jinzhu/configor"
	"github.com/jinzhu/gorm"
	"github.com/prometheus/common/log"
	"github.com/zdarovich/fibbage-game-server/db/models"
	"path/filepath"
	"strings"
)

var Seeds = struct {
	Final []struct {
		Category           string   `json:"category"`
		Question           string   `json:"question"`
		Answer             string   `json:"answer"`
		AlternateSpellings []string `json:"alternateSpellings"`
		Suggestions        []string `json:"suggestions"`
	} `json:"final"`
	Normal []struct {
		Category           string   `json:"category"`
		Question           string   `json:"question"`
		Answer             string   `json:"answer"`
		AlternateSpellings []string `json:"alternateSpellings"`
		Suggestions        []string `json:"suggestions"`
	} `json:"normal"`
}{}
func init() {
	filepaths, _ := filepath.Glob(filepath.Join("questions.json"))
	if err := configor.Load(&Seeds, filepaths...); err != nil {
		panic(err)
	}

}
func TruncateTables(db *gorm.DB,tables ...interface{}) {
	for _, table := range tables {
		if err := db.DropTableIfExists(table).Error; err != nil {
			panic(err)
		}

		db.AutoMigrate(table)
	}
}

func createQuestions(db *gorm.DB) {
	for _, p := range Seeds.Normal {
		q := models.Question{
			ModeType:           models.FACT,
			Category:           p.Category,
			Question:           strings.Replace(p.Question, "<BLANK>", "______", -1),
			Answer:             p.Answer,
			AlternateSpellings: strings.Join(p.AlternateSpellings, ","),
			//Suggestions:        strings.Join(p.Suggestions, ","),
		}
		if err := db.Create(&q).Error; err != nil {
		}
	}
}

func main() {
	connStr := fmt.Sprintf(
		"%s:%s@(%s)/fibbage_db?charset=utf8&parseTime=True&loc=Local",
		"db.user",
		"db.password",
		"db.host",
	)
	db, err := gorm.Open("mysql", connStr)
	if err != nil {
		panic(err)
	}
	var Tables = []interface{}{
		&models.Question{},
		//&models.QuestionAnswerEntity{},
		&models.Room{},
		//&models.Score{},
		//&models.User{},
	}
	TruncateTables(db,Tables...)
	log.Info("Start create sample data...")

	createQuestions(db)
	log.Info("--> Created questions.")
	
}
