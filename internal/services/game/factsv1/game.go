package factsv1

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/looplab/fsm"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	"github.com/zdarovich/fibbage-game-server/internal/db/models"
	"github.com/zdarovich/fibbage-game-server/internal/services"
	"github.com/zdarovich/fibbage-game-server/internal/services/game"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/event"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/fibbage"
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
		state     *fsm.FSM
		db        *gorm.DB
		groupUuid string
		done      chan struct{}
		players   map[string]*Player
	}
)

// New returns a Handler Base implementation
func New(groupUuid string, db *gorm.DB) *Game {
	return &Game{
		groupUuid: groupUuid,
		state:     fibbage.New(),
		done:      make(chan struct{}),
		players:   make(map[string]*Player),
		db:        db,
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

// Join room
func (r *Game) Join(ctx context.Context, msg *NicknameMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	if msg == nil || msg.Nickname == "" {
		return &Response{Code: 1, Result: "fail"}, nil
	} else if r.state != nil && r.state.Current() != state.WAITING {
		logger.Log.Infof("wrong state to join: %s", r.state.Current())
		return &Response{Code: 1, Result: r.state.Current()}, nil
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
	r.players[s.UID()].connected = true

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
		r.players[s.UID()].connected = false
		if count == 0 {
			r.reset()
			//pitaya.Shutdown()
		} else {
			pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onPlayerDisconnected", &User{UID: s.UID()})
		}
	})

	return &Response{Result: "success"}, nil
}

func (r *Game) reset() {
	r.state = fibbage.New()
	r.done = make(chan struct{})
	players := make(map[string]*Player)
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
}

func (r *Game) Stop(ctx context.Context, msg []byte) (*Response, error) {

	return &Response{Result: "success"}, nil
}

func (r *Game) Input(ctx context.Context, msg *InputMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	switch r.state.Current() {
	case state.INPUT_CATEGORY:

		if r.players[s.UID()].ready {
			return &Response{Result: "fail"}, nil
		}

		idx := msg.CategoryId
		if idx >= 0 && idx < len(r.players[s.UID()].categories) {
			r.players[s.UID()].categoryId = idx
			r.players[s.UID()].ready = true
			err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onReady",
				&User{
					UID: s.UID(),
				},
			)
			if err != nil {
				return nil, err
			}
			return &Response{Result: "success"}, nil

		}
		return &Response{Result: "fail"}, nil

	case state.INPUT_LIE_TEXT:
		if r.players[s.UID()].ready {
			return &Response{Result: "fail"}, nil
		} else if msg.Answer == "" {
			return &Response{Result: "fail"}, nil
		} else if s.UID() == GetCurrentPlayerId(r.players) && msg.Answer == r.players[s.UID()].question.Answer {
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

		return &Response{Result: "success"}, nil

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
			err := r.resetPlayerReadiness(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.changeCurrentPlayer(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.two(ctx)
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
			err := r.resetPlayerReadiness(ctx)
			if err != nil {
				errCh = err
				break loop
			}
			err = r.three(ctx)
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

			err = r.state.Event(event.START_FINISH)
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
			playersLeft := false
			for _, player := range r.players {
				if player.used {
					continue
				}
				playersLeft = true
				break
			}
			if playersLeft {
				err = r.state.Event(event.START_REPEAT)
				if err != nil {
					errCh = err
					break loop
				}
			} else {
				err = r.state.Event(event.START_RESET)
				if err != nil {
					errCh = err
					break loop
				}
			}
		case state.RESET:
			err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
				State: r.state.Current(),
			})
			if err != nil {
				errCh = err
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
	r.reset()
}

func (r *Game) starting(ctx context.Context) error {
	timeWait := 5

	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state.Current(),
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}

func (r *Game) one(ctx context.Context) error {
	for _, player := range r.players {
		player.ready = false
	}
	var categories []string
	r.db.Find(&[]models.Question{}).Pluck("category", &categories)
	members, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return err
	}
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
		r.players[uid].categories = choosenCategories
	}

	timeWait := 5

	for _, uid := range members {
		s := session.GetSessionByUID(uid)
		err = s.Push("onState", &Message{
			State:      r.state.Current(),
			Categories: r.players[uid].categories,
			Ticks:      timeWait,
		})
		if err != nil {
			return err
		}

	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}

func (r *Game) playerQuestion(ctx context.Context) error {

	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	for _, uid := range members {
		var question models.Question

		r.db.First(&question, "category = ?", r.players[uid].categories[r.players[uid].categoryId])

		r.players[uid].question = &Question{
			Question: question.Question,
			Answer:   question.Answer,
		}
	}
	timeWait := 5

	for _, uid := range members {
		s := session.GetSessionByUID(uid)

		err = s.Push("onState", &Message{
			State:    r.state.Current(),
			Question: r.players[uid].question,
			Ticks:    timeWait,
		})
		if err != nil {
			return err
		}
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}

func (r *Game) input(ctx context.Context) error {
	defer func() {
		logger.Log.Info("stop waiting for input")
	}()

	timeWait := 30
	err := pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State: r.state.Current(),
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
	for {
		select {
		case <-timeout:
			return nil
		case <-ticker.C:
			if ArePlayersReady(r.players, members) {
				return nil
			}
		}
	}

}

func (r *Game) changeCurrentPlayer(ctx context.Context) error {
	currentPlayerId := GetCurrentPlayerId(r.players)
	if _, ok := r.players[currentPlayerId]; ok {
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

	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}
	r.players[currentPlayerId].current = true
	r.players[currentPlayerId].used = true
	return nil
}

func (r *Game) two(ctx context.Context) error {

	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}
	other := &Question{
		Question: r.players[currentPlayerId].question.Question,
	}
	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	for _, uid := range members {
		r.players[uid].ready = false
	}
	timeWait := 5

	err = pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State:           r.state.Current(),
		Other:           other,
		CurrentPlayerId: currentPlayerId,
		Ticks:           timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)

	//time.Sleep(5 * time.Second)
	return nil
}

func (r *Game) resetPlayerReadiness(ctx context.Context) error {
	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	for _, uid := range members {
		r.players[uid].ready = false
	}
	return nil
}

func (r *Game) three(ctx context.Context) error {

	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}
	r.players[currentPlayerId].ready = true

	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	//lieAnswers := GetCurrentAnswers(r.players, currentPlayerId)
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
	err = pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State:           r.state.Current(),
		Answers:         lieAnswers,
		CurrentPlayerId: currentPlayerId,
		Ticks:           timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)

	//time.Sleep(5 * time.Second)

	return nil
}

func (r *Game) score(ctx context.Context) error {
	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}
	r.players[currentPlayerId].answerTruthId = 0
	r.players[currentPlayerId].current = false
	//shuffledAnswers := GetShuffledAnswers(GetCurrentAnswers(r.players, GetCurrentPlayerId(r.players)), r.unshuffled)

	//scoreMap := GetPlayersScore(r.players, currentPlayerId, shuffledAnswers)
	scoreMap := GetPlayersScoreV2(r.players, currentPlayerId)

	finalScore := make(map[string]int)
	for uid, score := range scoreMap {
		r.players[uid].totalScore = r.players[uid].totalScore + score
		finalScore[uid] = r.players[uid].totalScore
	}

	answermatrix := GetAnswersMatrix(r.players, currentPlayerId)

	members, err := pitaya.GroupMembers(ctx, r.groupUuid)
	if err != nil {
		return err
	}
	for _, uid := range members {
		r.players[uid].answerLie = ""
		r.players[uid].answerTruthId = 0
	}

	timeWait := 10
	err = pitaya.GroupBroadcast(ctx, "game", r.groupUuid, "onState", &Message{
		State:   r.state.Current(),
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
		State: r.state.Current(),
		Ticks: timeWait,
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(int64(timeWait)) * time.Second)
	return nil
}
