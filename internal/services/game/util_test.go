package game

import (
	"fmt"
	"github.com/bmizerany/assert"
	"github.com/zdarovich/fibbage-game-server/internal/db/models"
	"testing"
)

func TestGetPlayersScore1(t *testing.T) {
	currentPlayerId := "player1"
	playerTwoId := "player2"
	playerThreeId := "player3"
	expected := make(map[string]int)
	expected[currentPlayerId] = 0
	expected[playerTwoId] = 1000
	expected[playerThreeId] = 1000

	players := make(map[string]*Player, 3)
	players[currentPlayerId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer1",
		},
		answerLie: "answerLie1",
		ready:     true,
	}
	players[playerTwoId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer2",
		},
		answerLie:     "answerLie2",
		answerTruthId: 0,
		ready:         true,
	}
	players[playerThreeId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer3",
		},
		answerLie:     "answerLie3",
		answerTruthId: 0,
		ready:         true,
	}
	currentAnswers := GetCurrentAnswers(players, currentPlayerId)
	result := GetPlayersScore(players, currentPlayerId, currentAnswers)

	assert.Equal(t, expected, result)
}

func TestGetPlayersScore2(t *testing.T) {
	currentPlayerId := "player1"
	playerTwoId := "player2"
	playerThreeId := "player3"
	expected := make(map[string]int)
	expected[currentPlayerId] = 500
	expected[playerTwoId] = 1000
	expected[playerThreeId] = 0

	players := make(map[string]*Player, 3)
	players[currentPlayerId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer1",
		},
		answerLie: "answerLie1",
		ready:     true,
	}
	players[playerTwoId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer2",
		},
		answerLie:     "answerLie2",
		answerTruthId: 0,
		ready:         true,
	}
	players[playerThreeId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer3",
		},
		answerLie:     "answerLie3",
		answerTruthId: 1,
		ready:         true,
	}
	currentAnswers := GetCurrentAnswers(players, currentPlayerId)
	fmt.Println(currentAnswers)
	result := GetPlayersScore(players, currentPlayerId, currentAnswers)

	assert.Equal(t, expected, result)
}

func TestGetPlayersScore3(t *testing.T) {
	currentPlayerId := "player1"
	playerTwoId := "player2"
	playerThreeId := "player3"
	expected := make(map[string]int)
	expected[currentPlayerId] = 1000
	expected[playerTwoId] = 0
	expected[playerThreeId] = 0

	players := make(map[string]*Player, 3)
	players[currentPlayerId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer1",
		},
		answerLie: "answerLie1",
		ready:     true,
	}
	players[playerTwoId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer2",
		},
		answerLie:     "answerLie2",
		answerTruthId: 1,
		ready:         true,
	}
	players[playerThreeId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer3",
		},
		answerLie:     "answerLie3",
		answerTruthId: 1,
		ready:         true,
	}
	currentAnswers := GetCurrentAnswers(players, currentPlayerId)
	fmt.Println(currentAnswers)
	result := GetPlayersScore(players, currentPlayerId, currentAnswers)

	assert.Equal(t, expected, result)
}

func TestGetPlayersScore4(t *testing.T) {
	currentPlayerId := "player1"
	playerTwoId := "player2"
	playerThreeId := "player3"
	expected := make(map[string]int)
	expected[currentPlayerId] = 0
	expected[playerTwoId] = 500
	expected[playerThreeId] = 500

	players := make(map[string]*Player, 3)
	players[currentPlayerId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer1",
		},
		answerLie: "answerLie1",
		ready:     true,
	}
	players[playerTwoId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer2",
		},
		answerLie:     "answerLie2",
		answerTruthId: 3,
		ready:         true,
	}
	players[playerThreeId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer3",
		},
		answerLie:     "answerLie3",
		answerTruthId: 2,
		ready:         true,
	}
	currentAnswers := GetCurrentAnswers(players, currentPlayerId)
	fmt.Println(currentAnswers)
	result := GetPlayersScore(players, currentPlayerId, currentAnswers)

	assert.Equal(t, expected, result)
}

func TestGetPlayersScore5(t *testing.T) {
	currentPlayerId := "player1"
	playerTwoId := "player2"
	playerThreeId := "player3"
	expected := make(map[string]int)
	expected[currentPlayerId] = 0
	expected[playerTwoId] = 0
	expected[playerThreeId] = 1500

	players := make(map[string]*Player, 3)
	players[currentPlayerId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer1",
		},
		answerLie: "answerLie1",
		ready:     true,
	}
	players[playerTwoId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer2",
		},
		answerLie:     "answerLie2",
		answerTruthId: 3,
		ready:         true,
	}
	players[playerThreeId] = &Player{
		question: &models.Question{
			Answer: "truthAnswer3",
		},
		answerLie:     "answerLie3",
		answerTruthId: 0,
		ready:         true,
	}
	currentAnswers := GetCurrentAnswers(players, currentPlayerId)
	fmt.Println(currentAnswers)
	result := GetPlayersScore(players, currentPlayerId, currentAnswers)

	assert.Equal(t, expected, result)
}
