package services

import "github.com/zdarovich/fibbage-game-server/internal/db/models"

func Remove(s []string, i int) []string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func RemoveQuestions(s []models.Question, i int) []models.Question {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}
