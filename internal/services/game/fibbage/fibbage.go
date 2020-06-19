package fibbage

import (
	"github.com/looplab/fsm"
	"github.com/topfreegames/pitaya/logger"
	event2 "github.com/zdarovich/fibbage-game-server/internal/services/game/event"
	state2 "github.com/zdarovich/fibbage-game-server/internal/services/game/state"
)

func New() *fsm.FSM {
	return fsm.NewFSM(
		state2.WAITING,
		fsm.Events{
			{Name: event2.LAUNCH, Src: []string{state2.WAITING}, Dst: state2.STARTING},
			{Name: event2.START_ONE, Src: []string{state2.STARTING}, Dst: state2.ONE}, // choose category and read your question and answer
			{Name: event2.INPUT, Src: []string{state2.ONE}, Dst: state2.INPUT_CATEGORY},
			{Name: event2.START_SHOW_CHOICE, Src: []string{state2.INPUT_CATEGORY}, Dst: state2.SHOWING_CHOICE}, // show question to player
			{Name: event2.START_TWO, Src: []string{state2.SHOWING_CHOICE}, Dst: state2.TWO},                    // write lie answer for other player question
			{Name: event2.INPUT, Src: []string{state2.TWO}, Dst: state2.INPUT_LIE_TEXT},
			{Name: event2.START_THREE, Src: []string{state2.INPUT_LIE_TEXT}, Dst: state2.THREE}, // choose true answer for other player question
			{Name: event2.INPUT, Src: []string{state2.THREE}, Dst: state2.INPUT_TRUE_OPTION},
			{Name: event2.START_SCORE, Src: []string{state2.INPUT_TRUE_OPTION}, Dst: state2.SCORE}, // count score of given answers
			{Name: event2.START_FINISH, Src: []string{state2.SCORE}, Dst: state2.FINISH},           // finish game
			{Name: event2.START_REPEAT, Src: []string{state2.FINISH}, Dst: state2.TWO},             // repeat round two for remaining player questions
			{Name: event2.START_RESET, Src: []string{state2.FINISH}, Dst: state2.RESET},            // reset game
		},
		fsm.Callbacks{
			event2.INPUT: func(e *fsm.Event) {
				logger.Log.Info("INPUT event")
			},
			event2.LAUNCH: func(e *fsm.Event) {
				logger.Log.Info("LAUNCH event")
			},
			event2.START_ONE: func(e *fsm.Event) {
				logger.Log.Info("START_ONE event")
			},
			event2.START_SHOW_CHOICE: func(e *fsm.Event) {
				logger.Log.Info("START_SHOW_CHOICE event")
			},
			event2.START_TWO: func(e *fsm.Event) {
				logger.Log.Info("START_TWO event")
			},
			event2.START_THREE: func(e *fsm.Event) {
				logger.Log.Info("START_THREE event")
			},
			event2.START_SCORE: func(e *fsm.Event) {
				logger.Log.Info("START_SCORE event")
			},
			event2.START_REPEAT: func(e *fsm.Event) {
				logger.Log.Info("START_REPEAT event")
			},
			event2.START_FINISH: func(e *fsm.Event) {
				logger.Log.Info("START_FINISH event")
			},
			state2.STARTING: func(e *fsm.Event) {
				logger.Log.Info("STARTING state")
			},
			state2.ONE: func(e *fsm.Event) {
				logger.Log.Info("ONE state")
			},
			state2.TWO: func(e *fsm.Event) {
				logger.Log.Info("TWO state")
			},
			state2.SHOWING_CHOICE: func(e *fsm.Event) {
				logger.Log.Info("SHOWING_CHOICE state")
			},
			state2.THREE: func(e *fsm.Event) {
				logger.Log.Info("THREE state")
			},
			state2.SCORE: func(e *fsm.Event) {
				logger.Log.Info("SCORE state")
			},
			state2.INPUT_CATEGORY: func(e *fsm.Event) {
				logger.Log.Info("INPUT_CATEGORY state")
			},
			state2.INPUT_LIE_TEXT: func(e *fsm.Event) {
				logger.Log.Info("INPUT_LIE_TEXT state")
			},
			state2.INPUT_TRUE_OPTION: func(e *fsm.Event) {
				logger.Log.Info("INPUT_TRUE_OPTION state")
			},
		},
	)
}
