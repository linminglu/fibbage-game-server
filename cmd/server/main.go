package main

import (
	"context"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/spf13/viper"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/component"
	"github.com/topfreegames/pitaya/config"
	"github.com/topfreegames/pitaya/groups"
	"github.com/topfreegames/pitaya/serialize/json"
	"github.com/zdarovich/fibbage-game-server/internal/services/game/factsv2"
	"github.com/zdarovich/fibbage-game-server/pkg/acceptor"
	"strings"
)

func main() {

	defer pitaya.Shutdown()

	s := json.NewSerializer()
	conf := configApp()

	pitaya.SetSerializer(s)
	gsi := groups.NewMemoryGroupService(config.NewConfig(conf))
	pitaya.InitGroups(gsi)
	err := pitaya.GroupCreate(context.Background(), conf.GetString("group.uuid"))
	if err != nil {
		panic(err)
	}
	connStr := fmt.Sprintf(
		"%s:%s@(%s)/fibbage_db?charset=utf8&parseTime=True&loc=Local",
		conf.Get("db.user"),
		conf.Get("db.password"),
		conf.Get("db.host"),
	)
	db, err := gorm.Open("mysql", connStr)
	if err != nil {
		panic(err)
	}
	g := factsv2.New(conf.GetString("group.uuid"), db)
	pitaya.Register(g,
		component.WithName("game"),
		component.WithNameFunc(strings.ToLower),
	)
	//t := acceptor.NewWSAcceptor(":3250")
	t := acceptor.NewWSAcceptor(":3250")
	pitaya.AddAcceptor(t)

	pitaya.Configure(true, "game", pitaya.Cluster, map[string]string{}, conf)
	pitaya.Start()
}

func configApp() *viper.Viper {
	conf := viper.New()
	conf.SetDefault("pitaya.buffer.handler.localprocess", 15)
	conf.Set("pitaya.heartbeat.interval", "15s")
	conf.Set("pitaya.buffer.agent.messages", 32)
	conf.Set("pitaya.handler.messages.compression", false)
	conf.SetDefault("group.uuid", "game")
	conf.SetDefault("db.user", "newuser")
	conf.SetDefault("db.password", "password")
	conf.SetDefault("db.host", "localhost")
	return conf
}
