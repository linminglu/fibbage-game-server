package main

import (
	"context"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/acceptor"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/zdarovich/fibbage-game-server/services/game"
	"strings"
)

func main() {

	defer pitaya.Shutdown()

	s := json.NewSerializer()
	conf := configApp()

	pitaya.SetSerializer(s)
	gsi := groups.NewMemoryGroupService(config.NewConfig(conf))
	pitaya.InitGroups(gsi)
	err := pitaya.GroupCreate(context.Background(), conf.GetString("pitaya.group.name.uuid"))
	if err != nil {
		panic(err)
	}

	// Migrate the schema
	db, err := gorm.Open("mysql", "root:password@/fibbage_db?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		panic("failed to connect database")
	}

	g := game.New(conf.GetString("pitaya.group.name.uuid"), db)
	pitaya.Register(g,
		component.WithName("game"),
		component.WithNameFunc(strings.ToLower),
	)
	t := acceptor.NewWSAcceptor(":3250")
	pitaya.AddAcceptor(t)

	pitaya.Configure(true, "game", pitaya.Cluster, map[string]string{}, conf)
	pitaya.Start()
}

func configApp() *viper.Viper {
	conf := viper.New()
	conf.SetEnvPrefix("fibbage")
	conf.SetDefault("pitaya.buffer.handler.localprocess", 15)
	conf.Set("pitaya.heartbeat.interval", "15s")
	conf.Set("pitaya.buffer.agent.messages", 32)
	conf.Set("pitaya.handler.messages.compression", false)
	conf.SetDefault("pitaya.group.name.uuid", "game")
	return conf
}
