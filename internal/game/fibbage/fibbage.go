package fibbage

import (
	"github.com/looplab/fsm"
	"github.com/topfreegames/pitaya/logger"
	"github.com/zdarovich/fibbage-game-server/internal/game/event"
	"github.com/zdarovich/fibbage-game-server/internal/game/state"
)

func New() *fsm.FSM {
	return fsm.NewFSM(
		state.WAITING,
		fsm.Events{
			{Name: event.INPUT, Src: []string{state.ONE}, Dst: state.INPUT_CATEGORY},
			{Name: event.INPUT, Src: []string{state.TWO}, Dst: state.INPUT_LIE_TEXT},
			{Name: event.INPUT, Src: []string{state.THREE}, Dst: state.INPUT_TRUE_OPTION},
			{Name: event.LAUNCH, Src: []string{state.WAITING}, Dst: state.STARTING},
			{Name: event.START_ONE, Src: []string{state.STARTING}, Dst: state.ONE},  // choose category and read your question and answer
			{Name: event.START_TWO, Src: []string{state.ONE}, Dst: state.TWO},       // write lie answer for other player question
			{Name: event.START_THREE, Src: []string{state.TWO}, Dst: state.THREE},   // choose true answer for other player question
			{Name: event.START_SCORE, Src: []string{state.THREE}, Dst: state.SCORE}, // count score of given answers
			{Name: event.START_REPEAT, Src: []string{state.SCORE}, Dst: state.TWO},  // repeat round two for remaining player questions
			{Name: event.STOP, Src: []string{state.SCORE}, Dst: state.FINISH},       // finish game
		},
		fsm.Callbacks{
			event.INPUT: func(e *fsm.Event) {
				logger.Log.Info("INPUT event")
			},
			event.LAUNCH: func(e *fsm.Event) {
				logger.Log.Info("LAUNCH event")
			},
			event.START_ONE: func(e *fsm.Event) {
				logger.Log.Info("START_ONE event")
			},
			event.START_TWO: func(e *fsm.Event) {
				logger.Log.Info("START_TWO event")
			},
			event.START_THREE: func(e *fsm.Event) {
				logger.Log.Info("START_THREE event")
			},
			event.START_SCORE: func(e *fsm.Event) {
				logger.Log.Info("START_SCORE event")
			},
			event.START_REPEAT: func(e *fsm.Event) {
				logger.Log.Info("START_REPEAT event")
			},
			event.STOP: func(e *fsm.Event) {
				logger.Log.Info("STOP event")
			},
			state.STARTING: func(e *fsm.Event) {
				logger.Log.Info("STARTING state")
			},
			state.ONE: func(e *fsm.Event) {
				logger.Log.Info("ONE state")
			},
			state.TWO: func(e *fsm.Event) {
				logger.Log.Info("TWO state")
			},
			state.THREE: func(e *fsm.Event) {
				logger.Log.Info("THREE state")
			},
			state.SCORE: func(e *fsm.Event) {
				logger.Log.Info("SCORE state")
			},
			state.INPUT_CATEGORY: func(e *fsm.Event) {
				logger.Log.Info("INPUT_CATEGORY state")
			},
			state.INPUT_LIE_TEXT: func(e *fsm.Event) {
				logger.Log.Info("INPUT_LIE_TEXT state")
			},
			state.INPUT_TRUE_OPTION: func(e *fsm.Event) {
				logger.Log.Info("INPUT_TRUE_OPTION state")
			},
		},
	)
}
