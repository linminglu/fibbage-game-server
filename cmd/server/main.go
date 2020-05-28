package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zdarovich/fibbage-game-server/internal/db"
	"github.com/zdarovich/fibbage-game-server//internal/fibbage"
	"github.com/zdarovich/fibbage-game-server//internal/handler"
	"log"
	"net"
)

func runTCP(address string, hub signalr.HubInterface) {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Listening for TCP connection on %s\n", listener.Addr())

	server, _ := signalr.NewServer(signalr.UseHub(hub))

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Println(err)
			break
		}

		go server.Run(context.TODO(), tcp.NewNetConnection(conn))
	}
}

func main() {
	//go runTCP("127.0.0.1:8007", hub)
	game := handler.Game{}
	router := gin.Default()

	router.Use(gin.Recovery())
	router.LoadHTMLGlob("templates/*.html")
	router.Static("/public", "public")

	group := router.Group("/game")
	group.GET("/", game.GetGame)
	group.POST("/join", game.PostGameJoin)
	group.GET("/play", game.GetGamePlay)
	group.GET("/play/:uuid", game.GetGamePlay)

	h := &fibbage.Fibbage{}
	signalr.MapHubGin(router, "/fibbage", h)

	fmt.Printf("Listening for websocket connections on %s\n", "localhost:8086")

	if err := router.Run("localhost:8086"); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
