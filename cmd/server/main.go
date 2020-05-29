package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/zdarovich/fibbage-game-server/internal/game"
	"github.com/zdarovich/fibbage-game-server/internal/room"
	"log"
	"strings"
)



func main() {
	go func() {
		router := gin.Default()
		router.Use(gin.Recovery())
		router.Static("/public", "public")
		if err := router.Run("localhost:8086"); err != nil {
			log.Fatal("ListenAndServe:", err)
		}
	}()
	defer pitaya.Shutdown()

	s := json.NewSerializer()
	conf := configApp()

	pitaya.SetSerializer(s)
	gsi := groups.NewMemoryGroupService(config.NewConfig(conf))
	pitaya.InitGroups(gsi)
	err := pitaya.GroupCreate(context.Background(), "room")
	if err != nil {
		panic(err)
	}
	room := room.NewRoom()
	pitaya.Register(room,
		component.WithName("room"),
		component.WithNameFunc(strings.ToLower),
	)

	game := game.New()
	pitaya.Register(game,
		component.WithName("game"),
		component.WithNameFunc(strings.ToLower),
	)
	t := acceptor.NewWSAcceptor(":3250")
	pitaya.AddAcceptor(t)

	pitaya.Configure(true, "chat", pitaya.Cluster, map[string]string{}, conf)
	pitaya.Start()
}

func configApp() *viper.Viper {
	conf := viper.New()
	conf.SetEnvPrefix("chat") // allows using env vars in the CHAT_PITAYA_ format
	conf.SetDefault("pitaya.buffer.handler.localprocess", 15)
	conf.Set("pitaya.heartbeat.interval", "15s")
	conf.Set("pitaya.buffer.agent.messages", 32)
	conf.Set("pitaya.handler.messages.compression", false)
	return conf
}