package fibbage

import (
	"github.com/zdarovich/fibbage-game-server/internal/log"
	"github.com/zdarovich/fibbage-game-server/internal/signalr"
)

type Fibbage struct {
	signalr.Hub
}

func (c *Fibbage) OnConnected(connectionID string) {

	if uuid, ok := c.Items().Load("roomID"); ok {
		if uuid, ok := uuid.(string); ok {
			log.Infof("%s connected to group %s\n", connectionID, uuid)
			c.Groups().AddToGroup(uuid, connectionID)
			game := c.Games().LoadGame(uuid)
			if game == nil {
				log.Info("create new game")
				game = New(uuid, c.Clients())
				game.Init()
				c.Games().CreateGame(game)
			}
			game.OnConnected(connectionID)

		}
	}

}

func (c *Fibbage) OnDisconnected(connectionID string) {
	if uuid, ok := c.Items().Load("roomID"); ok {
		if uuid, ok := uuid.(string); ok {
			c.Groups().RemoveFromGroup(uuid, connectionID)
			log.Infof("%s disconnected from group %s\n", connectionID, uuid)
			game := c.Games().LoadGame(uuid)
			game.OnDisconnected(connectionID)

			if len(c.Clients().Group(uuid).ConnectionID()) == 0 {
				log.Infof("removed empty group %s\n", uuid)
				c.Groups().RemoveGroup(uuid)
				if game != nil {
					game.Stop()
					log.Infof("removed game %s\n", uuid)
					c.Games().RemoveGame(uuid)
				}
			}

		}
	}
}

func (c *Fibbage) OnInput(input string) {
	if uuid, ok := c.Items().Load("roomID"); ok {
		if uuid, ok := uuid.(string); ok {
			game := c.Games().LoadGame(uuid)
			if game != nil {
				connectionID := c.Clients().Caller().ConnectionID()[0]
				game.Input(connectionID, input)
			}
		}
	}
}

func (c *Fibbage) OnStart() {
	if uuid, ok := c.Items().Load("roomID"); ok {
		if uuid, ok := uuid.(string); ok {
			game := c.Games().LoadGame(uuid)
			if game != nil {
				game.Start()
			}
		}
	}
}
