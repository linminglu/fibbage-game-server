package game

import (
	"fmt"
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
		if p.answerLie == "" {
			p.answerLie = fmt.Sprintf("%s's lie", p.name) // if player missed answer in round 2 return random
		}

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

func GetPlayerIdByShuffledAnswerIdx(players map[string]*Player, shuffledAnswerIdx int) string {
	for uid, p := range players {
		if p.shuffledAnswerIdx == shuffledAnswerIdx {
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
		} else if !player.ready {
			continue // we don't count score for missed answer
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

func GetPlayersScoreV2(players map[string]*Player, currentPlayerId string) map[string]int {
	scoreMap := make(map[string]int)
	for uid, _ := range players {
		scoreMap[uid] = 0
	}
	currentTruthAnswerId := players[currentPlayerId].question.ShuffledAnswerIdx
	for uid, player := range players {
		if uid == currentPlayerId {
			continue // we dont count score for current player
		} else if !player.ready {
			continue // we don't count score for missed answer
		}
		if player.answerTruthId == currentTruthAnswerId {
			scoreMap[uid] = scoreMap[uid] + 1000
		} else {
			lyingPlayerId := GetPlayerIdByShuffledAnswerIdx(players, player.answerTruthId)
			if lyingPlayerId != "" {
				scoreMap[lyingPlayerId] = scoreMap[lyingPlayerId] + 500
			}
		}
	}
	return scoreMap
}

func GetAnswersMatrix(players map[string]*Player, currentPlayerId string) map[string]*AnswerMatrixRow {
	var result = make(map[string]*AnswerMatrixRow)

	for uid, _ := range players {
		result[uid] = &AnswerMatrixRow{}
	}
	result["truth"] = &AnswerMatrixRow{Text: strings.ToLower(players[currentPlayerId].question.Answer)}
	currentPlayerTruthAnswrIdx := players[currentPlayerId].question.ShuffledAnswerIdx
	for lUid, lyingPlayer := range players {
		result[lUid].Text = strings.ToLower(lyingPlayer.answerLie)

		for fUid, fooledPlayer := range players {
			if fUid == currentPlayerId {
				continue
			} else if !fooledPlayer.ready {
				continue // we don't count score for missed answer
			}
			if fooledPlayer.answerTruthId == lyingPlayer.shuffledAnswerIdx {
				result[lUid].PickedIds = append(result[lUid].PickedIds, fUid)
			}
		}
	}
	for uid, p := range players {
		if uid == currentPlayerId {
			continue
		} else if !p.ready {
			continue // we don't count score for missed answer
		}
		if p.answerTruthId == currentPlayerTruthAnswrIdx {
			result["truth"].PickedIds = append(result["truth"].PickedIds, uid)
		}

	}

	return result
}

func GetShuffledAnswers(shuffledAnswers []string, idxs []int) []string {
	var result = make([]string, len(idxs))
	for i, aidx := range idxs {
		result[aidx] = shuffledAnswers[i]
	}
	return result
}

func ArePlayersReady(players map[string]*Player, members []string) bool {
	for _, uid := range members {
		if players[uid].ready {
			continue
		} else {
			return false
		}
	}
	return true
}
