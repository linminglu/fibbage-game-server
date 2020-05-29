package game

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/looplab/fsm"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	"github.com/zdarovich/fibbage-game-server/internal/db"
	"github.com/zdarovich/fibbage-game-server/internal/errors"
	"github.com/zdarovich/fibbage-game-server/internal/game/event"
	"github.com/zdarovich/fibbage-game-server/internal/game/fibbage"
	"github.com/zdarovich/fibbage-game-server/internal/game/state"
	"github.com/zdarovich/fibbage-game-server/internal/log"
	"github.com/zdarovich/fibbage-game-server/internal/models"
	"math/big"
	"time"
)

type (
	// game represents a component that contains a bundle of room related handler
	// like Join/Message
	game struct {
		component.Base
		timer *timer.Timer
		state *fsm.FSM
		done  chan struct{}
	}

	// UserMessage represents a message that user sent
	StateMessage struct {
		State string `json:"state"`
	}

	// UserMessage represents a message that user sent
	StatusMessage struct {
		ServerTime string `json:"time,omitempty"`
		Message    string `json:"message,omitempty"`
	}

	// Response represents the result of joining room
	Response struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
)

// New returns a Handler Base implementation
func New() *game {
	return &game{state: fibbage.New(), done: make(chan struct{})}
}

// AfterInit component lifetime callback
func (r *game) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), "room")
		logger.Log.Debugf("UserCount: Time=> %s, Count=> %d, Error=> %q", time.Now().String(), count, err)
	})
}

func (r *game) Start(ctx context.Context, msg []byte) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	err := r.state.Event(event.LAUNCH)
	if err != nil {
		if err := s.Push("onError", &errors.Error{
			Code:    errors.EVENT_FAILED,
			Message: "start event failed",
		}); err != nil {
			return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "sending error"})
		}
		return &Response{Result: "fail"}, nil
	}

	go r.loop()

	return &Response{Result: "success"}, nil
}

func (r *game) Stop(ctx context.Context, msg []byte) (*Response, error) {

	return &Response{Result: "success"}, nil
}

func (r *game) Input(ctx context.Context, msg []byte) (*Response, error) {

	return &Response{Result: "success"}, nil
}

func (r *game) loop() {
	logger.Log.Info("start loop")
	ctx := context.Background()
	var errCh error
loop:
	for {
		select {
		case <-r.done:
			break loop
		default:
		}
		err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &StateMessage{
			State: r.state.Current(),
		})
		if err != nil {
			errCh = err
			break loop
		}
		switch r.state.Current() {
		case state.STARTING:
			err := r.starting(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.START_ONE)
			if err != nil {
				errCh = err
				break loop
			}
		case state.ONE:
			err := r.one(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.INPUT)
			if err != nil {
				errCh = err
				break loop
			}
		case state.INPUT_CATEGORY:
			err := r.category(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.START_TWO)
			if err != nil {
				errCh = err
				break loop
			}
		case state.TWO:
			err := r.two(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.INPUT)
			if err != nil {
				errCh = err
				break loop
			}
		case state.INPUT_LIE_TEXT:
			err := r.lie(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.START_THREE)
			if err != nil {
				errCh = err
				break loop
			}
		case state.THREE:
		case state.SCORE:
		case state.FINISH:
			break loop
		default:
			logger.Log.Errorf("unknown state %s", r.state.Current())
			break loop
		}
	}
	if errCh != nil {
		logger.Log.Error(errCh)
	}
	logger.Log.Info("stop loop")
}

func (r *game) starting(ctx context.Context) error {
	startTime := 5
	for i := 0; i < startTime; i++ {
		select {
		case <-time.After(1 * time.Second):
			err := pitaya.GroupBroadcast(ctx, "game", "game", "OnStatus",
				&StatusMessage{
					ServerTime: time.Now().String(),
					Message:    fmt.Sprintf("starting game in %d seconds", startTime-i),
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *game) one(ctx context.Context) error {
	var categories []string
	db.DB.Find(&[]models.Question{}).Pluck("category", &categories)
	members, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return err
	}
	categoriesMap := make(map[string][]string)
	for _, uid := range members {
		logger.Log.Infof("assign 5 random categories to user %s", uid)
		var choosenCategories []string
		for i := 0; i < 5; i++ {
			categoriesCount := len(categories)
			logger.Log.Infof("categories left %d", categoriesCount)
			var ri int64
			randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(categoriesCount)))
			if err != nil {
				logger.Log.Error(err)
				ri = 0
			} else {
				ri = randIdx.Int64()
			}
			choosenCategories = append(choosenCategories, categories[ri])
			categories = remove(categories, int(ri))
		}
		categoriesMap[uid] = choosenCategories
	}

	for uid, categories := range categoriesMap {
		s := session.GetSessionByUID(uid)
		err := s.Push("OnChoice", categories)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *game) category(ctx context.Context) error {
	roundTime := 10
	for i := 0; i < roundTime; i++ {
		select {
		case <-time.After(1 * time.Second):
			err := pitaya.GroupBroadcast(ctx, "game", "game", "OnStatus",
				&StatusMessage{
					ServerTime: time.Now().String(),
					Message:    fmt.Sprintf("seconds left for answer %d", roundTime-i),
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *game) two(ctx context.Context) error {
	return nil
}

func (r *game) lie(ctx context.Context) error {
	return nil
}
