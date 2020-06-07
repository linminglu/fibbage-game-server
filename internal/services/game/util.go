package game

import (
	"strings"
)

func GetUsedIcons(players map[string]*Player) []string {
	var res []string
	for _, p := range players {
		res = append(res, p.iconName)
	}
	return res
}

func GetCurrentAnswers(players map[string]*Player, currentPlayerId string) []string {
	var res []string
	res = append(res, strings.ToLower(players[currentPlayerId].question.Answer))
	for _, p := range players {
		res = append(res, strings.ToLower(p.answerLie))
	}

	return res
}
func GetCurrentPlayerId(players map[string]*Player) string {
	for uid, p := range players {
		if p.current {
			return uid
		}
	}
	return ""
}
func GetPlayerIdByLieAnswer(players map[string]*Player, answer string) string {
	for uid, p := range players {
		if strings.ToLower(p.answerLie) == strings.ToLower(answer) {
			return uid
		}
	}
	return ""
}

func GetPlayersScore(players map[string]*Player, currentPlayerId string, currentAnswers []string) map[string]int {
	scoreMap := make(map[string]int)
	for uid, _ := range players {
		scoreMap[uid] = 0
	}
	currentPlayerTruth := players[currentPlayerId].question.Answer
	for uid, player := range players {
		if uid == currentPlayerId {
			continue // we dont count score for current player
		}
		answer := currentAnswers[player.answerTruthId]
		if strings.ToLower(answer) == strings.ToLower(currentPlayerTruth) {
			scoreMap[uid] = scoreMap[uid] + 1000
		} else {
			lyingPlayerId := GetPlayerIdByLieAnswer(players, answer)
			if lyingPlayerId != "" {
				scoreMap[lyingPlayerId] = scoreMap[lyingPlayerId] + 500
			}
		}
	}
	return scoreMap
}

func ArePlayersReady(players map[string]*Player) bool {
	for _, p := range players {
		if p.ready {
			continue
		} else {
			return false
		}
	}
	return true
}
