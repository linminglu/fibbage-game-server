package fibbage

import (
	"crypto/rand"
	"fmt"
	"github.com/looplab/fsm"
	"github.com/zdarovich/fibbage-game-server/internal/db"
	"github.com/zdarovich/fibbage-game-server/internal/log"
	"github.com/zdarovich/fibbage-game-server/internal/models"
	"github.com/zdarovich/fibbage-game-server/internal/signalr"
	"math/big"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

const (
	WAIT         string = "WAIT"
	START        string = "START"
	STOP         string = "STOP"
	SCORE        string = "SCORE"
	INPUT_OPTION string = "INPUT_OPTION"
	INPUT_TEXT   string = "INPUT_TEXT"
	ONE          string = "ONE"
	TWO          string = "TWO"
	THREE        string = "THREE"
	FINISH       string = "FINISH"
)

type (
	game struct {
		gameID     string
		hubClients signalr.HubClients
		userDatas  sync.Map
		state      *fsm.FSM
	}
	userData struct {
		Name        string
		Score       int
		Categories  []string
		CategoryIdx int
		Ready       bool
		Answer      string
	}
	readyMap struct {
		ready map[string]bool
		lock  sync.RWMutex
	}
)

func (g *game) OnConnected(connectionID string) {
	if g.state == nil {
		return
	}
	var serverUser models.User
	db.DB.Where("connection_id = ?", connectionID).First(&serverUser)

	g.userDatas.Store(connectionID, userData{
		Name:  serverUser.Name,
		Score: 0,
	})
	g.userDatas.Store("test", userData{
		Name:  "testuser",
		Score: 0,
	})
	var users []User
	g.userDatas.Range(func(key, value interface{}) bool {
		if ud, ok := value.(userData); ok {
			users = append(users, User{
				ID:   key.(string),
				Name: ud.Name,
			})
		}
		return true
	})
	g.hubClients.Group(g.gameID).Send("OnPlayerConnected", users)
	g.hubClients.Group(g.gameID).Send("OnStatus", Status{
		ID:         g.gameID,
		State:      g.state.Current(),
		ServerTime: time.Now().String(),
		Message:    "player connected",
	})
}

func (g *game) OnDisconnected(connectionID string) {
	if g.state == nil {
		return
	}
	g.userDatas.Delete(connectionID)
	var users []User
	g.userDatas.Range(func(key, value interface{}) bool {
		if ud, ok := value.(userData); ok {
			users = append(users, User{
				ID:   key.(string),
				Name: ud.Name,
			})
		}
		return true
	})
	g.hubClients.Group(g.gameID).Send("OnPlayerConnected", users)
	g.hubClients.Group(g.gameID).Send("OnStatus", Status{
		ID:         g.gameID,
		State:      g.state.Current(),
		ServerTime: time.Now().String(),
		Message:    "player disconnected",
	})
}

func (g *game) Input(connectionID string, input string) {
	if g.state == nil {
		return
	}
	if ud, ok := g.userDatas.Load(connectionID); ok {
		if ud, ok := ud.(userData); ok {
			if ud.Ready {
				g.hubClients.Client(connectionID).Send("OnInputResponse", Error{ID: connectionID, Message: "input already sent"})
			}
			switch g.state.Current() {
			case INPUT_OPTION:
				idx, err := strconv.Atoi(input)
				if err != nil {
					g.hubClients.Client(connectionID).Send("OnInputResponse", Error{ID: connectionID, Message: "input must be option number"})
					return
				}
				if idx > 0 && idx < len(ud.Categories) {
					ud.CategoryIdx = idx
					ud.Ready = true
					g.userDatas.Store(connectionID, ud)
					g.hubClients.Group(g.gameID).Send("OnPlayerReady", Ready{ID: connectionID, Ready: true})
				} else {
					g.hubClients.Client(connectionID).Send("OnInputResponse", Error{ID: connectionID, Message: "input must be option number"})
				}

			case INPUT_TEXT:

				if len(input) == 0 {
					g.hubClients.Client(connectionID).Send("OnInputResponse", Error{ID: connectionID, Message: "input must not be empty"})
					return
				}
				ud.Answer = input
				ud.Ready = true
				g.userDatas.Store(connectionID, ud)
				g.hubClients.Group(g.gameID).Send("OnPlayerReady", Ready{ID: connectionID, Ready: true})
			default:
				g.hubClients.Client(connectionID).Send("OnInputResponse", Error{ID: connectionID, Message: "not input state"})
			}
		}
	}

}

func (g *game) Start() {
	if g.state == nil || g.state.Current() != WAIT {
		return
	}
	go func() {
		err := g.state.Event(START)
		if err != nil {
			g.hubClients.Group(g.gameID).Send("OnStatus", Status{
				ID:         g.gameID,
				State:      g.state.Current(),
				ServerTime: time.Now().String(),
				Error:      err.Error(),
			})
			return
		}
		err = g.state.Event(ONE)
		if err != nil {
			g.hubClients.Group(g.gameID).Send("OnStatus", Status{
				ID:         g.gameID,
				State:      g.state.Current(),
				ServerTime: time.Now().String(),
				Error:      err.Error(),
			})
			return
		}
		err = g.state.Event(INPUT_OPTION)
		if err != nil {
			g.hubClients.Group(g.gameID).Send("OnStatus", Status{
				ID:         g.gameID,
				State:      g.state.Current(),
				ServerTime: time.Now().String(),
				Error:      err.Error(),
			})
			return
		}

		err = g.state.Event(TWO)
		if err != nil {
			g.hubClients.Group(g.gameID).Send("OnStatus", Status{
				ID:         g.gameID,
				State:      g.state.Current(),
				ServerTime: time.Now().String(),
				Error:      err.Error(),
			})
			return
		}
	}()

}

func (g *game) Stop() {
	if g.state == nil {
		return
	}
	g.state.SetState(STOP)
}

func (g *game) recover() {
	if err := recover(); err != nil {
		log.Error(err)
		log.Error(string(debug.Stack()))
		log.Info("recover panic")
		g.userDatas.Range(func(key, value interface{}) bool {
			if ud, ok := value.(userData); ok {
				ud.Score = 0
				ud.CategoryIdx = 0
				ud.Categories = nil
			}
			return true
		})
		g.state.SetState(WAIT)
		g.hubClients.Group(g.gameID).Send("OnStatus", Status{
			ID:         g.gameID,
			State:      g.state.Current(),
			ServerTime: time.Now().String(),
			Message:    "recovered from game error",
		})
	}
}

func (g *game) GameID() string {
	return g.gameID
}

func (g *game) Init() {
	g.state = fsm.NewFSM(
		WAIT,
		fsm.Events{
			{Name: START, Src: []string{WAIT}, Dst: START},
			{Name: ONE, Src: []string{START}, Dst: ONE},
			{Name: INPUT_OPTION, Src: []string{ONE}, Dst: INPUT_OPTION},
			{Name: TWO, Src: []string{INPUT_OPTION}, Dst: TWO},
			{Name: INPUT_TEXT, Src: []string{TWO}, Dst: INPUT_TEXT},
			{Name: THREE, Src: []string{INPUT_TEXT}, Dst: THREE},
			{Name: FINISH, Src: []string{THREE}, Dst: FINISH},
		},
		fsm.Callbacks{
			"START": func(e *fsm.Event) {
				log.Info("starting game")
				defer g.recover()
				startTime := 5
				for i := 0; i < startTime; i++ {
					select {
					case <-time.After(1 * time.Second):
						g.hubClients.Group(g.gameID).Send("OnStatus", Status{
							ID:         g.gameID,
							State:      g.state.Current(),
							ServerTime: time.Now().String(),
							Message:    fmt.Sprintf("starting game in %d seconds", startTime-i),
						})
					}
				}
			},

			"ONE": func(e *fsm.Event) {
				log.Info("round one")
				defer g.recover()
				g.hubClients.Group(g.gameID).Send("OnStatus", Status{
					ID:         g.gameID,
					State:      g.state.Current(),
					ServerTime: time.Now().String(),
					Message:    "round one",
				})
				log.Info("get categories from db")
				var categories []string
				db.DB.Find(&[]models.Question{}).Pluck("category", &categories)

				for _, connectionID := range g.hubClients.Group(g.gameID).ConnectionID() {
					if ud, ok := g.userDatas.Load(connectionID); ok {
						if ud, ok := ud.(userData); ok {

							log.Infof("assign 5 random categories to user %s", connectionID)
							var choosenCategories []string
							for i := 0; i < 5; i++ {
								categoriesCount := len(categories)
								log.Infof("categories left %d", categoriesCount)
								var ri int64
								randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(categoriesCount)))
								if err != nil {
									log.Error(err)
									ri = 0
								} else {
									ri = randIdx.Int64()
								}
								choosenCategories = append(choosenCategories, categories[ri])
								categories = remove(categories, int(ri))
							}
							ud.Categories = choosenCategories
							ud.Ready = false
							g.userDatas.Store(connectionID, ud)
						}
					}
				}
				g.userDatas.Range(func(key, value interface{}) bool {
					if ud, ok := value.(userData); ok {
						log.Infof("%s - %+v", key, ud)
						log.Infof("send 5 random categories to user %s", key)
						g.hubClients.Client(key.(string)).Send("OnStatus", Status{
							ID:         g.gameID,
							State:      g.state.Current(),
							ServerTime: time.Now().String(),
							Message:    "choose categories",
						})
						g.hubClients.Client(key.(string)).Send("OnChoice", ud.Categories)
					}
					return true
				})
			},

			"INPUT_OPTION": func(e *fsm.Event) {
				log.Info("wait for player input")

				defer g.recover()
				roundTime := 5
				for i := 0; i < roundTime; i++ {
					select {
					case <-time.After(1 * time.Second):
						g.hubClients.Group(g.gameID).Send("OnStatus", Status{
							ID:         g.gameID,
							State:      g.state.Current(),
							ServerTime: time.Now().String(),
							Message:    fmt.Sprintf("seconds left for answer %d", roundTime-i),
						})
					}
				}
				log.Info("stop waiting input")
			},

			"TWO": func(e *fsm.Event) {
				log.Info("round two")
				defer g.recover()

				g.hubClients.Group(g.gameID).Send("OnStatus", Status{
					ID:         g.gameID,
					State:      g.state.Current(),
					ServerTime: time.Now().String(),
					Message:    "round two",
				})
				//TODO get question from db by category
				//TODO assign to player data
				//TODO send questions to players
				//TODO set timer for 10 sec
				//TODO shuffle player ids
				//TODO send player question to all players
			},
			"INPUT_TEXT": func(e *fsm.Event) {
				log.Info("wait for player input")

				defer g.recover()
				roundTime := 5
				for i := 0; i < roundTime; i++ {
					select {
					case <-time.After(1 * time.Second):
						g.hubClients.Group(g.gameID).Send("OnStatus", Status{
							ID:         g.gameID,
							State:      g.state.Current(),
							ServerTime: time.Now().String(),
							Message:    fmt.Sprintf("seconds left for answer %d", roundTime-i),
						})
					}
				}
				log.Info("stop waiting input")
			},
			"SCORE": func(e *fsm.Event) {
				log.Info("round three")
				defer g.recover()

				g.hubClients.Group(g.gameID).Send("OnStatus", Status{
					ID:         g.gameID,
					State:      g.state.Current(),
					ServerTime: time.Now().String(),
					Message:    "round three",
				})
				//TODO count scoring for answers
				//TODO send player score
			},
			"FINISH": func(e *fsm.Event) {
				log.Info("round three")
				defer g.recover()

				g.hubClients.Group(g.gameID).Send("OnStatus", Status{
					ID:         g.gameID,
					State:      g.state.Current(),
					ServerTime: time.Now().String(),
					Message:    "round three",
				})
				//TODO count final score for answers
				//TODO send final score
				//TODO send winner announcement to players
			},
		},
	)
}

func New(roomID string, hubClients signalr.HubClients) signalr.Game {

	return &game{
		hubClients: hubClients,
		gameID:     roomID,
	}
}
