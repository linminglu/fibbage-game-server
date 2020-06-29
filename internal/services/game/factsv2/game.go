package factsv2

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	"github.com/zdarovich/fibbage-game-server/internal/db/models"
	"github.com/zdarovich/fibbage-game-server/internal/services"
	"github.com/zdarovich/fibbage-game-server/internal/services/game"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/state"
	"math/big"
	mathRand "math/rand"
	"time"
)

type (
	// Game represents a component that contains a bundle of room related handler
	// like Join/Status
	Game struct {
		component.Base
		timer     *timer.Timer
		db        *gorm.DB
		groupUuid string
		done      chan struct{}
		state     string
		players   map[string]*Player
	}
)

// New returns a Handler Base implementation
func New(groupUuid string, db *gorm.DB) *Game {
	return &Game{
		groupUuid: groupUuid,
		done:      make(chan struct{}),
		players:   make(map[string]*Player),
		db:        db,
		state:     state.WAITING,
	}
}

// AfterInit component lifetime callback
func (r *Game) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), r.groupUuid)
		logger.Log.Debugf("UserCount: Time=> %s, Count=> %d, Error=> %q", time.Now().String(), count, err)
	})
}

func (r *Game) Start(ctx context.Context, msg []byte) (*Response, error) {

	if r.state != state.WAITING {
		return &Response{Code: 1, Result: "fail"}, nil
	}

	go r.loop()

	return &Response{Result: "success"}, nil
}

// Join room
func (r *Game) Join(ctx context.Context, msg *NicknameMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	if msg == nil || msg.Nickname == "" {
		return &Response{Result: "fail"}, nil
	} else if r.state != state.WAITING {
		logger.Log.Infof("wrong state to join: %s", r.state)
		return &Response{Result: "fail"}, nil
	}

	err := s.Bind(ctx, uuid.New().String()) // binding session uid

	if err != nil {
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	err = pitaya.GroupAddMember(ctx, r.groupUuid, s.UID()) // add session to group
	if err != nil {
		return nil, err
	}
	r.players[s.UID()] = &Player{}
	r.players[s.UID()].name = msg.Nickname

	uids, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return nil, err
	}
	usedIcons := make(map[string]bool)
	for _, p := range r.players {
		usedIcons[p.iconName] = true
	}
	tempIcons := make([]string, 0)
	for _, i := range game.IconSet {
		if usedIcons[i] {
			continue
		}
		tempIcons = append(tempIcons, i)
	}
	iconsCount := len(tempIcons)
	var ri int64
	randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(iconsCount)))
	if err != nil {
		logger.Log.Error(err)
		ri = 0
	} else {
		ri = randIdx.Int64()
	}
	r.players[s.UID()].iconName = tempIcons[ri]

	var users []User
	for _, uid := range uids {
		if uid == s.UID() {
			users = append(users, User{
				UID:      uid,
				Name:     r.players[uid].name,
				Icon:     r.players[uid].iconName,
				IsPlayer: true,
			})
		} else {
			users = append(users, User{
				UID:  uid,
				Name: r.players[uid].name,
				Icon: r.players[uid].iconName,
			})
		}

	}
	for _, uid := range uids {
		if uid == s.UID() {
			err := s.Push("onCreatePlayer", users)
			if err != nil {
				return nil, err
			}
			continue
		}
		sess := session.GetSessionByUID(uid)
		err := sess.Push("onCreatePlayer", []User{{
			UID:  s.UID(),
			Name: r.players[s.UID()].name,
			Icon: r.players[s.UID()].iconName,
		}})
		if err != nil {
			return nil, err
		}
	}

	// on session close, remove it from group
	s.OnClose(func() {
		pitaya.GroupRemoveMember(ctx, r.groupUuid, s.UID())
		count, _ := pitaya.GroupCountMembers(context.Background(), r.groupUuid)
		if count == 0 {
			r.reset()
		} else {
			pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onPlayerDisconnected", &User{UID: s.UID()})
		}
	})

	return &Response{Code: 1, Result: "success"}, nil
}

func (r *Game) reset() {
	r.done = make(chan struct{})
	r.players = make(map[string]*Player)
	r.state = state.WAITING
}

func (r *Game) restart() {
	r.done = make(chan struct{})
	players := make(map[string]*Player)
	r.state = state.WAITING
	for uid, p := range r.players {
		resetPlayer := &Player{
			name:              p.name,
			question:          nil,
			categories:        nil,
			categoryId:        0,
			totalScore:        0,
			answerLie:         "",
			shuffledAnswerIdx: 0,
			answerTruthId:     0,
			iconName:          p.iconName,
			ready:             false,
			used:              false,
			current:           false,
			connected:         true,
		}
		players[uid] = resetPlayer
	}
	r.players = players
	_ = pitaya.GroupBroadcast(context.Background(), "game", r.groupUuid, "onState", &Message{
		State: r.state,
	})
}

func (r *Game) Stop(ctx context.Context, msg []byte) (*Response, error) {

	return &Response{Code: 1, Result: "success"}, nil
}

func (r *Game) Input(ctx context.Context, msg *InputMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	switch r.state {
	case state.INPUT_LIE_TEXT:
		if r.players[s.UID()].ready {
			return &Response{Result: "fail"}, nil
		} else if msg.Answer == "" {
			return &Response{Result: "fail"}, nil
		}
		r.players[s.UID()].answerLie = msg.Answer

		r.players[s.UID()].ready = true
		err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onReady",
			&User{
				UID: s.UID(),
			},
		)
		if err != nil {
			return nil, err
		}

		return &Response{Code: 1, Result: "success"}, nil

	case state.INPUT_TRUE_OPTION:
		if r.players[s.UID()].ready {
			return &Response{Result: "fail"}, nil
		}
		idx := msg.AnswerId
		if idx >= 0 && idx < len(r.players)+1 { // each player lie answer + 1 truth answer
			if idx == r.players[s.UID()].shuffledAnswerIdx {
				return &Response{Result: "fail"}, nil
			}
			r.players[s.UID()].answerTruthId = idx

			r.players[s.UID()].ready = true

			err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onReady",
				&User{
					UID: s.UID(),
				},
			)
			if err != nil {
				return nil, err
			}
			return &Response{Code: 1, Result: "success"}, nil
		}

	default:
		logger.Log.Errorf("wrong state to input %s", r.state)
		return &Response{Result: "fail"}, nil
	}
	return &Response{Result: "fail"}, nil
}

func (r *Game) loop() error {
	logger.Log.Info("start loop")
	ctx := context.Background()
	err := r.starting(ctx)
	if err != nil {
		return err
	}
	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	err = r.one(ctx, members)
	if err != nil {
		return err
	}
	for i := 0; i < len(members); i++ {

		err = r.two(ctx, members[i])
		if err != nil {
			return err
		}
		r.state = state.INPUT_LIE_TEXT
		err = r.input(ctx)
		if err != nil {
			return err
		}
		err = r.three(ctx, members, members[i])
		if err != nil {
			return err
		}
		r.state = state.INPUT_TRUE_OPTION
		err = r.input(ctx)
		if err != nil {
			return err
		}
		r.state = state.SCORE
		err = r.score(ctx, members, members[i])
		if err != nil {
			return err
		}
		r.state = state.FINISH
		err := r.finish(ctx)
		if err != nil {
			return err
		}
	}

	logger.Log.Info("stop loop")
	r.restart()
	return nil
}

func (r *Game) starting(ctx context.Context) error {
	r.state = state.STARTING
	timeWait := 5

	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state,
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}

func (r *Game) one(ctx context.Context, members []string) error {
	var questions []models.Question
	r.db.Where("lang_code = ?", "ru").Find(&questions)
	if len(questions) == 0 {
		return errors.New("no questions")
	}
	for _, uid := range members {
		questionsCount := len(questions)
		var ri int64
		randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(questionsCount)))
		if err != nil {
			logger.Log.Error(err)
			ri = 0
		} else {
			ri = randIdx.Int64()
		}
		r.players[uid].question = &Question{
			Question: questions[int(ri)].Question,
			Answer:   questions[int(ri)].Answer,
		}
		questions = services.RemoveQuestions(questions, int(ri))
	}

	return nil
}

func (r *Game) input(ctx context.Context) error {
	defer func() {
		logger.Log.Info("stop waiting for input")
	}()

	timeWait := 30
	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state,
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	logger.Log.Info("start waiting for input")

	ticker := time.NewTicker(1 * time.Second)
	timeout := time.After(time.Duration(int64(timeWait)) * time.Second)
	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
loop:
	for {
		select {
		case <-r.done:
			return errors.New("interupted")
		case <-timeout:
			break loop
		case <-ticker.C:
			if ArePlayersReady(r.players, members) {
				break loop
			}
		}
	}
	err = r.resetPlayerReadiness(ctx, members)
	if err != nil {
		return err
	}
	return nil
}

func (r *Game) changeCurrentPlayer(ctx context.Context) error {
	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId != "" {
		r.players[currentPlayerId].current = false
	}

	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	for _, uid := range members {
		if r.players[uid].used {
			continue
		}
		currentPlayerId = uid
		break
	}

	if currentPlayerId != "" {
		r.players[currentPlayerId].used = true
		r.players[currentPlayerId].current = true
	}

	return nil
}

func (r *Game) two(ctx context.Context, currentPlayerId string) error {

	other := &Question{
		Question: r.players[currentPlayerId].question.Question,
	}

	timeWait := 5

	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state,
		Other: other,
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)

	return nil
}

func (r *Game) resetPlayerReadiness(ctx context.Context, members []string) error {
	for _, uid := range members {
		r.players[uid].ready = false
	}
	return nil
}

func (r *Game) three(ctx context.Context, members []string, currentPlayerId string) error {

	type AnswerShuffled struct {
		Text string
		Id   string
	}
	var lieAnswersShuffled []*AnswerShuffled
	for _, uid := range members {
		answer := r.players[uid].answerLie
		if answer == "" {
			answer = fmt.Sprintf("%s's lie", r.players[uid].name) // if player missed answer in round 2 return random
		}
		lieAnswersShuffled = append(lieAnswersShuffled, &AnswerShuffled{
			Text: answer,
			Id:   uid,
		})
	}
	lieAnswersShuffled = append(lieAnswersShuffled, &AnswerShuffled{
		Text: r.players[currentPlayerId].question.Answer,
		Id:   "truth",
	})
	mathRand.Seed(time.Now().UnixNano())
	mathRand.Shuffle(len(lieAnswersShuffled), func(i, j int) {
		lieAnswersShuffled[i], lieAnswersShuffled[j] = lieAnswersShuffled[j], lieAnswersShuffled[i]
	})
	var lieAnswers []string
	for i := 0; i < len(lieAnswersShuffled); i++ {
		if lieAnswersShuffled[i].Id == "truth" {
			r.players[currentPlayerId].question.ShuffledAnswerIdx = i
		} else {
			r.players[lieAnswersShuffled[i].Id].shuffledAnswerIdx = i
		}
		lieAnswers = append(lieAnswers, lieAnswersShuffled[i].Text)
	}
	timeWait := 5
	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State:   r.state,
		Answers: lieAnswers,
		Ticks:   timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)

	//time.Sleep(5 * time.Second)

	return nil
}

func (r *Game) score(ctx context.Context, members []string, currentPlayerId string) error {
	scoreMap := GetPlayersScoreV2(r.players, currentPlayerId)

	finalScore := make(map[string]int)
	for uid, score := range scoreMap {
		r.players[uid].totalScore = r.players[uid].totalScore + score
		finalScore[uid] = r.players[uid].totalScore
	}

	answermatrix := GetAnswersMatrix(r.players, currentPlayerId)

	for _, uid := range members {
		r.players[uid].answerLie = ""
		r.players[uid].answerTruthId = 0
	}

	timeWait := 10
	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State:   r.state,
		Score:   scoreMap,
		Total:   finalScore,
		Choices: answermatrix,
		Ticks:   timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	//time.Sleep(10 * time.Second)
	return nil
}

func (r *Game) finish(ctx context.Context) error {
	timeWait := 5

	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state,
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}
