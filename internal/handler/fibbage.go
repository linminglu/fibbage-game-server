package handler

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/zdarovich/fibbage-game-server/internal/db"
	"github.com/zdarovich/fibbage-game-server/internal/models"
	"github.com/zdarovich/fibbage-game-server/internal/session"
	"net/http"
)

type (
	Game struct {
		DB *gorm.DB
	}
)

func (r *Game) PostGameJoin(c *gin.Context) {
	name := c.PostForm("name")
	roomUuid := c.PostForm("uuid")
	if name == "" {
		c.Redirect(http.StatusMovedPermanently, "/game")
		c.Abort()
		return
	}
	sess, err := session.Store.Get(c.Request, session.FIBBAGE_SESSION)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Set some session values.
	sess.Values[session.FIBBAGE_PLAYER_NAME] = name
	// Save it before we write to the response/return from the handler.
	err = session.Store.Save(c.Request, c.Writer, sess)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	path := ""
	if roomUuid != "" {
		path = fmt.Sprintf("/game/play/%s", roomUuid)
	} else {
		path = "/game/play"
	}
	c.Redirect(http.StatusMovedPermanently, path)
	c.Abort()
}

func (r *Game) GetGamePlay(c *gin.Context) {
	roomUuid := c.Param("uuid")

	sess, err := session.Store.Get(c.Request, session.FIBBAGE_SESSION)
	if err != nil {
		c.Redirect(http.StatusMovedPermanently, "/game")
		c.Abort()
		return
	}

	var nickname string
	if name, ok := sess.Values[session.FIBBAGE_PLAYER_NAME].(string); !ok {
		path := ""
		if roomUuid != "" {
			path = fmt.Sprintf("/game?uuid=%s", roomUuid)
		} else {
			path = "/game"
		}
		c.Redirect(http.StatusMovedPermanently, path)
		c.Abort()
		return
	} else {
		nickname = name
	}

	shareLink := ""
	if len(roomUuid) == 0 {
		roomUuid = uuid.New().String()
		shareLink = fmt.Sprintf("%s%s/%s", c.Request.Host, c.Request.URL.String(), roomUuid)
		c.Request.URL.Path = fmt.Sprintf("%s/%s", c.Request.URL.String(), roomUuid)
	}

	connectionID := getConnectionID()

	room := models.Room{Uuid: roomUuid}
	db.DB.Where(models.Room{Uuid: roomUuid}).FirstOrInit(&room)

	user := models.User{
		Name:         nickname,
		ConnectionID: connectionID,
		Room:         room,
	}
	db.DB.Save(&user)

	data := struct {
		Id   string
		Uuid string
		Link string
	}{
		Id:   connectionID,
		Uuid: roomUuid,
		Link: shareLink,
	}
	c.HTML(http.StatusOK, "game.html", data)
}

func (r *Game) GetGame(c *gin.Context) {
	roomUuid := c.Query("uuid")

	data := struct {
		Uuid string
	}{
		Uuid: roomUuid,
	}
	c.HTML(http.StatusOK, "index.html", data)
}

func getConnectionID() string {
	bytes := make([]byte, 16)
	// rand.Read only fails when the systems random number generator fails. Rare case, ignore
	_, _ = rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}
