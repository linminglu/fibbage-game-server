package main

import (
	"github.com/jinzhu/configor"
	"github.com/prometheus/common/log"
	"github.com/zdarovich/fibbage-game-server/db"
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
func TruncateTables(tables ...interface{}) {
	for _, table := range tables {
		if err := db.DB.DropTableIfExists(table).Error; err != nil {
			panic(err)
		}

		db.DB.AutoMigrate(table)
	}
}

func createQuestions() {
	for _, p := range Seeds.Normal {
		q := models.Question{
			ModeType:           models.FACT,
			Category:           p.Category,
			Question:           strings.Replace(p.Question, "<BLANK>", "______", -1),
			Answer:             p.Answer,
			AlternateSpellings: strings.Join(p.AlternateSpellings, ","),
			//Suggestions:        strings.Join(p.Suggestions, ","),
		}
		if err := db.DB.Create(&q).Error; err != nil {
		}
	}
}

func main() {
	var Tables = []interface{}{
		&models.Question{},
		//&models.QuestionAnswerEntity{},
		&models.Room{},
		//&models.Score{},
		//&models.User{},
	}
	TruncateTables(Tables...)
	log.Info("Start create sample data...")

	createQuestions()
	log.Info("--> Created questions.")
	
}
