package game

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/looplab/fsm"
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
		players        map[string]*Player
		currentAnswers []string
	}
	Instance struct {
		state          *fsm.FSM
		done           chan struct{}
		players        map[string]*Player
		currentAnswers []string
	}
	Player struct {
		name          string
		question      *models.Question
		categories    []string
		categoryId    int
		totalScore    int
		answerLie     string
		answerTruthId int
		iconName      string
		ready         bool
		used          bool
		current       bool
	}
)

// New returns a Handler Base implementation
func New() *Game {
	return &Game{
		state:   fibbage.New(),
		done:    make(chan struct{}),
		players: make(map[string]*Player),
	}
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

// Join room
func (r *Game) Join(ctx context.Context, msg *NicknameMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	if msg == nil || msg.Nickname == "" {
		return &Response{Code: 1, Result: "fail"}, nil
	} else if r.state != nil && r.state.Current() != state.WAITING {
		return &Response{Code: 1, Result: "fail"}, nil
	}

	err := s.Bind(ctx, uuid.New().String()) // binding session uid

	if err != nil {
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}
	err = pitaya.GroupAddMember(ctx, "game", s.UID()) // add session to group
	if err != nil {
		return nil, err
	}
	r.players[s.UID()] = &Player{}
	r.players[s.UID()].name = msg.Nickname

	uids, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return nil, err
	}
	usedIcons := make(map[string]bool)
	for _, p := range r.players {
		usedIcons[p.iconName] = true
	}
	tempIcons := make([]string, 0)
	for _, i := range iconSet {
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
		pitaya.GroupRemoveMember(ctx, "game", s.UID())
		delete(r.players, s.UID())
		count, _ := pitaya.GroupCountMembers(context.Background(), "game")
		if count == 0 {
			//r.done <- struct{}{}
		} else {
			pitaya.GroupBroadcast(ctx, "game", "game", "onPlayerDisconnected", &User{UID: s.UID()})
		}
	})

	return &Response{Result: "success"}, nil
}

func (r *Game) reset() {
	r.state = fibbage.New()
	r.done = make(chan struct{})
	for uid, p := range r.players {
		resetPlayer := &Player{
			name:          p.name,
			question:      nil,
			categories:    nil,
			categoryId:    0,
			totalScore:    0,
			answerLie:     "",
			answerTruthId: 0,
			iconName:      p.iconName,
			ready:         false,
			used:          false,
			current:       false,
		}
		r.players[uid] = resetPlayer
	}
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
			err := pitaya.GroupBroadcast(ctx, "game", "game", "onReady",
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
		err := pitaya.GroupBroadcast(ctx, "game", "game", "onReady",
			&Message{UserReady: &UserReady{UID: s.UID(), Ready: true}},
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
		if idx >= 0 && idx < len(r.currentAnswers) { // each player lie answer + 1 truth answer
			if r.currentAnswers[idx] == r.players[s.UID()].answerLie {
				return &Response{Result: "fail"}, nil
			}
			r.players[s.UID()].answerTruthId = idx

			r.players[s.UID()].ready = true

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
			err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
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
	startTime := 5

	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State: r.state.Current(),
	})
	if err != nil {
		return err
	}
	//startTime := 1
	for i := 0; i < startTime; i++ {
		select {
		case <-time.After(1 * time.Second):

		}
	}
	return nil
}

func (r *Game) one(ctx context.Context) error {
	for _, player := range r.players {
		player.ready = false
	}
	var categories []string
	db.DB.Find(&[]models.Question{}).Pluck("category", &categories)
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

	for uid, player := range r.players {
		s := session.GetSessionByUID(uid)

		err = s.Push("onState", &Message{
			State:      r.state.Current(),
			Categories: player.categories,
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
	for _, uid := range members {
		var question models.Question

		db.DB.First(&question, "category = ?", r.players[uid].categories[r.players[uid].categoryId])

		r.players[uid].question = &question

		s := session.GetSessionByUID(uid)

		err = s.Push("onState", &Message{
			State:    r.state.Current(),
			Question: r.players[uid].question,
		})
		if err != nil {
			return err
		}
	}

	roundTime := 5
	for i := 0; i < roundTime; i++ {
		select {
		case <-time.After(1 * time.Second):

		}
	}
	return nil
}

func (r *Game) input(ctx context.Context) error {
	defer func() {
		logger.Log.Info("stop waiting for input")
	}()

	roundTime := 30
	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State: r.state.Current(),
	})
	if err != nil {
		return err
	}
	logger.Log.Info("start waiting for input")

	for i := 0; i < roundTime; i++ {
		select {
		case <-time.After(1 * time.Second):

			if ArePlayersReady(r.players) {
				return nil
			}
		}
	}
	for _, p := range r.players {
		p.ready = true
	}

	return nil
}

func (r *Game) two(ctx context.Context) error {
	var currentPlayerId string
	for uid, player := range r.players {
		if player.used {
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

	other := &Question{
		Question: r.players[currentPlayerId].question.Question,
	}
	for _, player := range r.players {
		player.ready = false
	}
	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State:           r.state.Current(),
		Other:           other,
		CurrentPlayerId: currentPlayerId,
	})
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)
	return nil
}

func (r *Game) three(ctx context.Context) error {

	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}

	for uid, player := range r.players {
		if !player.ready {
			player.answerLie = fmt.Sprintf("%s's lie", player.name) // if player missed answer in round 2 return random
		}
		if uid == currentPlayerId {
			player.ready = true
		} else {
			player.ready = false
		}
	}
	lieAnswers := GetCurrentAnswers(r.players, currentPlayerId)

	mathRand.Seed(time.Now().UnixNano())
	mathRand.Shuffle(len(lieAnswers), func(i, j int) { lieAnswers[i], lieAnswers[j] = lieAnswers[j], lieAnswers[i] })
	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State:           r.state.Current(),
		Answers:         lieAnswers,
		CurrentPlayerId: currentPlayerId,
	})
	r.currentAnswers = lieAnswers
	if err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	return nil
}

func (r *Game) score(ctx context.Context) error {
	currentPlayerId := GetCurrentPlayerId(r.players)
	if currentPlayerId == "" {
		return errors.New("used player uid not left")
	}
	r.players[currentPlayerId].answerTruthId = 0
	r.players[currentPlayerId].current = false

	finalScore := make(map[string]int)

	scoreMap := GetPlayersScore(r.players, currentPlayerId, r.currentAnswers)

	for uid, score := range scoreMap {
		r.players[uid].totalScore = r.players[uid].totalScore + score
		finalScore[uid] = r.players[uid].totalScore
	}

	answermatrix := []map[string]string{
		{"num": ""}, {"num": "player1"}, {"num": "player2"}, {"num": "player3"},
		{"num": "player1"}, {"num": ""}, {"num": "fdfd"}, {"num": "fsdsdf"},
		{"num": "player2"}, {"num": "das"}, {"num": "das"}, {"num": "das"},
		{"num": "player3"}, {"num": "das"}, {"num": "das"}, {"num": "das"},
	}
	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State:   r.state.Current(),
		Score:   scoreMap,
		Total:   finalScore,
		Choices: answermatrix,
	})
	if err != nil {
		return err
	}

	time.Sleep(10 * time.Second)
	return nil
}

func (r *Game) finish(ctx context.Context) error {

	err := pitaya.GroupBroadcast(ctx, "game", "game", "onState", &Message{
		State: r.state.Current(),
	})
	if err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return nil
}
