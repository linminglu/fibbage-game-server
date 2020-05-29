package room

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/timer"
	"github.com/zdarovich/fibbage-game-server/internal/errors"
	"time"
)

type (
	// Room represents a component that contains a bundle of room related handler
	// like Join/Message
	Room struct {
		component.Base
		timer *timer.Timer
	}

	// NicknameMessage represents a message that user sent
	NicknameMessage struct {
		Nickname string `json:"nickname"`
	}

	// NewUser message will be received when new user join room
	User struct {
		UID  string `json:"id,omitempty"`
		Name string `json:"name"`
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
	return &Room{}
}

// AfterInit component lifetime callback
func (r *Room) AfterInit() {
	r.timer = pitaya.NewTimer(time.Minute, func() {
		count, err := pitaya.GroupCountMembers(context.Background(), "room")
		logger.Log.Debugf("UserCount: Time=> %s, Count=> %d, Error=> %q", time.Now().String(), count, err)
	})
}

// Join room
func (r *Room) Join(ctx context.Context, msg []byte) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)
	err := s.Bind(ctx, uuid.New().String()) // binding session uid

	if err != nil {
		return nil, pitaya.Error(err, "RH-000", map[string]string{"failed": "bind"})
	}

	// notify others
	pitaya.GroupBroadcast(ctx, "chat", "room", "onPlayerConnected", &User{UID: s.UID()})
	// new user join group
	pitaya.GroupAddMember(ctx, "room", s.UID()) // add session to group

	// on session close, remove it from group
	s.OnClose(func() {
		pitaya.GroupRemoveMember(ctx, "room", s.UID())
		pitaya.GroupBroadcast(ctx, "chat", "room", "onPlayerDisconnected", &User{UID: s.UID()})

	})

	return &Response{Result: "success"}, nil
}

// Message sync last message to all members
func (r *Room) CreateName(ctx context.Context, msg *NicknameMessage) (*Response, error) {
	s := pitaya.GetSessionFromCtx(ctx)

	if msg == nil || msg.Nickname == "" {
		if err := s.Push("onError", &errors.Error{
			Code:    errors.EMPTY_FIELD,
			Message: "nickname is empty",
		}); err != nil {
			fmt.Println("error push", err)
			return nil, err
		} else {
			return &Response{Result: "fail"}, nil
		}
	}
	err := s.Set("name", msg.Nickname)
	if err != nil {
		logger.Log.Error(err)
		return nil, err
	}
	err = pitaya.GroupBroadcast(ctx, "chat", "room", "onInputNameResponse", msg)
	if err != nil {
		logger.Log.Error(err)
		return nil, err
	}
	return &Response{Result: "success"}, nil
}
