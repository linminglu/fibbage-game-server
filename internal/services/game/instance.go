package game

import (
	"github.com/looplab/fsm"
)

type (
	Instance struct {
		state          *fsm.FSM
		done           chan struct{}
		players        map[string]*Player
		currentAnswers []string
	}
)
