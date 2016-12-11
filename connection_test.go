package ws

import (
	"testing"

	"log"

	"time"

	"github.com/gin-gonic/gin"
)

var connid = "testconn"
var wsHub *Hub

func TestSendClose(t *testing.T) {

	r := gin.Default()

	wsHub = NewHub()

	r.GET("/ws", func(c *gin.Context) {

		conn, _ := NewConn(connid, wsHub, c.Writer, c.Request)

		wsHub.Register(conn)

		go func() {

			time.Sleep(time.Second * 2)

			conn.Close()

			log.Println("AAAAA")
			err := conn.Send("aaaaaaa")
            log.Println("bbbbbb")
		}()

		for {

			msg, err := conn.Recv()
			if err != nil {
				break
			}

			log.Println(msg)
		}

		wsHub.UnRegister(conn)

	})

	r.Run(":10000")

}
