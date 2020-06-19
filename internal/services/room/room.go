package room

import (
	"context"
	"crypto/rand"
	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/session"
	"github.com/topfreegames/pitaya/timer"
	services2 "github.com/zdarovich/fibbage-game-server/internal/services"
	game2 "github.com/zdarovich/fibbage-game-server/internal/services/game"
	"math/big"
	"time"
)

type (
	// Room represents a component that contains a bundle of room related handler
	// like Join/Status
	Room struct {
		component.Base
		timer *timer.Timer
		icons map[string]string
	}

	// NicknameMessage represents a message that user sent
	NicknameMessage struct {
		Nickname string `json:"nickname"`
	}

	// NewUser message will be received when new user join room
	User struct {
		UID  string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
		Icon string `json:"icon,omitempty"`
	}

	// AllMembers contains all members uid
	AllMembers struct {
		Members []string `json:"members"`
	}

	// Response represents the result of joining room
	Response struct {
		Code   int    `json:"code"`
		Result string `json:"result"`
	}
)

// NewRoom returns a Handler Base implementation
func NewRoom() *Room {
	return &Room{icons: make(map[string]string)}
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), "game")
		logger.Log.Debugf("UserCount: Time=> %s, Count=> %d, Error=> %q", time.Now().String(), count, err)
	})
}

// Join room
func (r *Room) Join(ctx context.Context, msg *NicknameMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	if msg == nil || msg.Nickname == "" {
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

	err = s.Set(game2.NAME, msg.Nickname)
	if err != nil {
		logger.Log.Error(err)
		return nil, err
	}
	uids, err := pitaya.GroupMembers(ctx, "game")
	if err != nil {
		return nil, err
	}
	usedIcons := make(map[string]bool)
	for _, uid := range uids {
		usedIcons[r.icons[uid]] = true
	}
	tempIcons := make([]string, 0)
	for _, i := range iconSet {
		if !usedIcons[i] {
			tempIcons = append(tempIcons, i)
		}
	}
	for _, uid := range uids {
		iconsCount := len(tempIcons)
		var ri int64
		randIdx, err := rand.Int(rand.Reader, big.NewInt(int64(iconsCount)))
		if err != nil {
			logger.Log.Error(err)
			ri = 0
		} else {
			ri = randIdx.Int64()
		}
		r.icons[uid] = tempIcons[ri]
		tempIcons = services2.Remove(tempIcons, int(ri))
	}
	var users []User
	for _, uid := range uids {
		sess := session.GetSessionByUID(uid)
		if name, ok := sess.Get(game2.NAME).(string); ok {
			users = append(users, User{
				UID:  uid,
				Name: name,
				Icon: r.icons[uid],
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
			UID:  uid,
			Name: msg.Nickname,
		}})
		if err != nil {
			return nil, err
		}
	}

	// on session close, remove it from group
	s.OnClose(func() {
		pitaya.GroupRemoveMember(ctx, "game", s.UID())
		pitaya.GroupBroadcast(ctx, "game", "game", "onPlayerDisconnected", &User{UID: s.UID()})

	})

	return &Response{Result: "success"}, nil
}
