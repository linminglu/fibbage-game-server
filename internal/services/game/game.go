package game

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/looplab/fsm"
	"github.com/prometheus/common/log"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	"github.com/zdarovich/fibbage-game-server/internal/db"
	"github.com/zdarovich/fibbage-game-server/internal/db/models"
	"github.com/zdarovich/fibbage-game-server/internal/services"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/event"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/fibbage"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/state"
	"math/big"
	mathRand "math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

type (
	// Game represents a component that contains a bundle of room related handler
	// like Join/Status
	Game struct {
		component.Base
		timer          *timer.Timer
		state          *fsm.FSM
		done           chan struct{}
		ready          int32
		usedQuestions  map[string]bool
		currentPlayer  string
		currentAnswers []string
		finalScore     map[string]int
		dataMap        map[string]map[string]string
	}

	Message struct {
		CurrentPlayerId string              `json:"currentPlayerId,omitempty"`
		Ticks           int                 `json:"ticks,omitempty"`
		State           string              `json:"state,omitempty"`
		Categories      []string            `json:"categories,omitempty"`
		Answers         []string            `json:"answers,omitempty"`
		ServerTime      string              `json:"time,omitempty"`
		Status          string              `json:"status,omitempty"`
		Question        *models.Question    `json:"question,omitempty"`
		UserReady       *UserReady          `json:"userReady,omitempty"`
		Score           map[string]int      `json:"score,omitempty"`
		Total           map[string]int      `json:"total,omitempty"`
		Choices         []map[string]string `json:"answerMatrix,omitempty"`
	}
	UserReady struct {
		UID   string `json:"id,omitempty"`
		Ready bool   `json:"ready,omitempty"`
	}

	InputMessage struct {
		CategoryId int    `json:"categoryId,omitempty"`
		Answer     string `json:"answer,omitempty"`
		AnswerId   int    `json:"answerId,omitempty"`
	}

	// Response represents the result of joining room
	Response struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
)

// New returns a Handler Base implementation
func New() *Game {
	return &Game{state: fibbage.New(), done: make(chan struct{}), ready: 0, usedQuestions: make(map[string]bool), finalScore: make(map[string]int), dataMap: make(map[string]map[string]string)}
}

// AfterInit component lifetime callback
func (r *Game) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), "game")
		logger.Log.Debugf("UserCount: Time=> %s, Count=> %d, Error=> %q", time.Now().String(), count, err)
	})
}

func (r *Game) Start(ctx context.Context, msg []byte) (*Response, error) {

	err := r.state.Event(event.LAUNCH)
	if err != nil {
		return &Response{Code: 1, Result: "fail"}, nil
	}

	//uids, err := pitaya.GroupMembers(ctx, "game")
	//if err != nil {
	//	return &Response{Code: 1, Result: "fail"}, nil
	//}
	//for _, uid := range uids {
	//	s := session.GetSessionByUID(uid)
	//	if name, ok := s.Get(NAME).(string); ok {
	//		if name == "" {
	//			return &Response{Code: 1, Result: "fail"}, nil
	//		}
	//	}
	//}

	go r.loop()

	return &Response{Result: "success"}, nil
}

func (r *Game) Stop(ctx context.Context, msg []byte) (*Response, error) {

	return &Response{Result: "success"}, nil
}

func (r *Game) Input(ctx context.Context, msg *InputMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	switch r.state.Current() {
	case state.INPUT_CATEGORY:
		categoryId := s.Get(CATEGORYID)
		if categoryId != nil {
			return &Response{Result: "fail"}, nil
		}

		if categories, ok := s.Get(CATEGORIES).([]string); ok {
			idx := msg.CategoryId
			if idx >= 0 && idx < len(categories) {
				err := s.Set(CATEGORYID, idx)
				if err != nil {
					return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "set categoryid error"})
				}
				atomic.AddInt32(&r.ready, 1)

				err = pitaya.GroupBroadcast(ctx, "game", "game", "onReady",
					&Message{UserReady: &UserReady{UID: s.UID(), Ready: true}},
				)
				if err != nil {
					return nil, err
				}
				return &Response{Result: "success"}, nil

			} else {
				idx = 0
				err := s.Set(CATEGORYID, idx)
				if err != nil {
					return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "set categoryid error"})
				}
				return &Response{Result: "fail"}, nil
			}

		} else {
			return nil, pitaya.Error(errors.New("categories cast error"), "RH-000", map[string]string{"failed": "categories cast error"})
		}

	case state.INPUT_LIE_TEXT:
		answer := r.dataMap[s.UID()][ANSWER_LIE]
		if answer != "" {
			return &Response{Result: "fail"}, nil
		}
		if msg.Answer == "" {
			return &Response{Result: "fail"}, nil
		}
		r.dataMap[s.UID()][ANSWER_LIE] = msg.Answer

		atomic.AddInt32(&r.ready, 1)
		err := pitaya.GroupBroadcast(ctx, "game", "game", "onReady",
			&Message{UserReady: &UserReady{UID: s.UID(), Ready: true}},
		)
		if err != nil {
			return nil, err
		}

		return &Response{Result: "success"}, nil

	case state.INPUT_TRUE_OPTION:
		answerId := r.dataMap[s.UID()][ANSWERID]
		if answerId != "" {
			return &Response{Result: "fail"}, nil
		}
		idx := msg.AnswerId
		if idx >= 0 && idx < len(r.currentAnswers) {
			r.dataMap[s.UID()][ANSWERID] = strconv.Itoa(idx)

			atomic.AddInt32(&r.ready, 1)

			err := pitaya.GroupBroadcast(ctx, "game", "game", "onReady",
				&Message{UserReady: &UserReady{UID: s.UID(), Ready: true}},
			)
			if err != nil {
				return nil, err
			}
			return &Response{Result: "success"}, nil
		}

	default:
		logger.Log.Errorf("wrong state to input %s", r.state.Current())
		return &Response{Result: "fail"}, nil
	}
	return &Response{Result: "fail"}, nil
}

func (r *Game) loop() {
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
			err := r.input(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.START_SHOW_CHOICE)
			if err != nil {
				errCh = err
				break loop
			}
		case state.SHOWING_CHOICE:
			err := r.playerQuestion(ctx)
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
			err := r.input(ctx)
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
			err := r.three(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.INPUT)
			if err != nil {
				errCh = err
				break loop
			}
		case state.INPUT_TRUE_OPTION:
			err := r.input(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.START_SCORE)
			if err != nil {
				errCh = err
				break loop
			}
		case state.SCORE:
			err := r.score(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.state.Event(event.STOP)
			if err != nil {
				errCh = err
				break loop
			}
		case state.FINISH:
			err := r.finish(ctx)
			if err != nil {
				errCh = err
				break loop
			}

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
	r.state = fibbage.New()
}

func (r *Game) starting(ctx context.Context) error {
	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State: r.state.Current(),
	})
	if err != nil {
		return err
	}
	startTime := 5
	//startTime := 1
	for i := 0; i < startTime; i++ {
		select {
		case <-time.After(1 * time.Second):

		}
	}
	return nil
}

func (r *Game) one(ctx context.Context) error {

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
			categories = services.Remove(categories, int(ri))
		}
		categoriesMap[uid] = choosenCategories
	}

	for uid, categories := range categoriesMap {
		s := session.GetSessionByUID(uid)
		err := s.Set(CATEGORIES, categories)
		if err != nil {
			return err
		}
		err = s.Push("onCategories", &Message{
			Categories: categories,
		})
		if err != nil {
			return err
		}
	}
	time.Sleep(5 * time.Second)
	return nil
}

func (r *Game) playerQuestion(ctx context.Context) error {

	members, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return err
	}
	questionsMap := make(map[string]*models.Question)
	var question models.Question
	for _, uid := range members {
		s := session.GetSessionByUID(uid)
		if categories, ok := s.Get(CATEGORIES).([]string); ok {
			if categoryId, ok := s.Get(CATEGORYID).(int); ok {
				db.DB.First(&question, "category = ?", categories[categoryId])
			} else {
				logger.Log.Info("picking 0 question")
				db.DB.First(&question, "category = ?", categories[0])
			}
		} else {
			db.DB.First(&question)
		}
		questionsMap[uid] = &question
	}
	for uid, question := range questionsMap {
		s := session.GetSessionByUID(uid)

		err = s.Set(QUESTION, question)
		if _, ok := r.dataMap[uid]; ok {
			r.dataMap[uid][ANSWER_TRUTH] = question.Answer
		} else {
			r.dataMap[uid] = make(map[string]string)
			r.dataMap[uid][ANSWER_TRUTH] = question.Answer
		}
		r.usedQuestions[uid] = false

		if err != nil {
			return err
		}
		err := s.Push("onQuestion", &Message{Question: question})
		if err != nil {
			return err
		}
	}

	roundTime := 5
	for i := 0; i < roundTime; i++ {
		select {
		case <-time.After(1 * time.Second):
			err := pitaya.GroupBroadcast(ctx, "game", "game", "onStatus",
				&Message{
					ServerTime: time.Now().String(),
					Status:     fmt.Sprintf("seconds left for next round %d", roundTime-i),
				})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Game) input(ctx context.Context) error {
	defer func() {
		logger.Log.Info("stop waiting for input")
		r.ready = 0
	}()
	count, err := pitaya.GroupCountMembers(ctx, "game")
	if err != nil {
		return err
	}
	roundTime := 30
	logger.Log.Info("start waiting for input")

	for i := 0; i < roundTime; i++ {
		select {
		case <-time.After(1 * time.Second):
			err := pitaya.GroupBroadcast(ctx, "game", "game", "onStatus",
				&Message{
					ServerTime: time.Now().String(),
					Status:     fmt.Sprintf("seconds left for answer %d", roundTime-i),
				})
			if err != nil {
				return err
			}
			if int(r.ready) == count {
				return nil
			}
		}
	}
	return nil
}

func (r *Game) two(ctx context.Context) error {

	for uid, used := range r.usedQuestions {
		if used {
			continue
		}
		s := session.GetSessionByUID(uid)
		if question, ok := s.Get(QUESTION).(*models.Question); ok {
			question.Answer = ""
			question.AlternateSpellings = ""
			question.Suggestions = ""
			log.Info(uid)
			r.currentPlayer = uid
			r.usedQuestions[uid] = true

			err := pitaya.GroupBroadcast(ctx, "game", "game", "onQuestion", &Message{Question: question})
			if err != nil {
				return err
			}
			err = pitaya.GroupBroadcast(ctx, "game", "game", "onCurrentPlayer", &Message{CurrentPlayerId: uid})
			if err != nil {
				return err
			}
			break
		}
	}
	time.Sleep(5 * time.Second)
	return nil
}

func (r *Game) three(ctx context.Context) error {

	members, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return err
	}
	log.Infof("%+v", r.dataMap)
	var answers []string
	for _, uid := range members {
		if answer, ok := r.dataMap[uid][ANSWER_LIE]; ok {
			answers = append(answers, answer)
		}
	}
	log.Info(members)

	log.Info(r.currentPlayer)

	answers = append(answers, r.dataMap[r.currentPlayer][ANSWER_TRUTH])

	mathRand.Seed(time.Now().UnixNano())
	mathRand.Shuffle(len(answers), func(i, j int) { answers[i], answers[j] = answers[j], answers[i] })
	r.currentAnswers = answers
	err = pitaya.GroupBroadcast(ctx, "game", "game", "onAnswers",
		&Message{
			Answers: answers,
		})
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)

	return nil
}

func (r *Game) score(ctx context.Context) error {

	members, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return err
	}
	scoreMap := make(map[string]int)
	choiceMap := make(map[string]int)
	for _, uid := range members {
		answerId, err := strconv.Atoi(r.dataMap[uid][ANSWERID])
		if err != nil {
			logger.Log.Error(err)
			answerId = 0
		}
		choiceMap[uid] = answerId
		currentQuestionAnswer := r.dataMap[r.currentPlayer][ANSWER_TRUTH]

		if r.currentAnswers[answerId] == currentQuestionAnswer {
			scoreMap[uid] = scoreMap[uid] + 1000
		}

		for key, val := range r.dataMap {
			if key == uid {
				continue
			}
			if r.currentAnswers[answerId] == val[ANSWER_LIE] {
				scoreMap[key] = scoreMap[key] + 500
			}
		}
		r.finalScore[uid] = r.finalScore[uid] + scoreMap[uid]
	}
	err = pitaya.GroupBroadcast(ctx, "game", "game", "onScore",
		&Message{
			Score: scoreMap,
			Total: r.finalScore,
		})
	if err != nil {
		return err
	}

	answermatrix := []map[string]string{
		{"num": ""}, {"num": "player1"}, {"num": "player2"}, {"num": "player3"},
		{"num": "player1"}, {"num": ""}, {"num": "fdfd"}, {"num": "fsdsdf"},
		{"num": "player2"}, {"num": "das"}, {"num": "das"}, {"num": "das"},
		{"num": "player3"}, {"num": "das"}, {"num": "das"}, {"num": "das"},
	}
	err = pitaya.GroupBroadcast(ctx, "game", "game", "onChoices",
		&Message{
			Choices: answermatrix,
		})
	if err != nil {
		return err
	}

	time.Sleep(10 * time.Second)
	return nil
}

func (r *Game) finish(ctx context.Context) error {

	return nil
}
