package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func (c *Client) readPump() {
	defer func() {
		log.Println("[WS Hub] WS用户结束")
		c.hub.unregister <- c
		c.conn.Close()
	}()
	log.Println("[WS Hub] 用户处理启动")
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS Hub] WS连接发生未知错误: %v", err)
			}
			break
		}
	}
}

func newHub() *Hub {
	return &Hub{
		roomId:     -1,
		timeStamp:  time.Now().Unix(),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	//var ii = 0
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Println("[Danmaku Hub]", h.roomId, "新用户")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Println("[Danmaku Hub]", h.roomId, "用户断开")
				if len(h.clients) <= 0 {
					log.Println("[Danmaku Hub]", h.roomId, "用户为0，关闭直播间监听")
					delete(ACConnMap, h.roomId)
					delete(ACWatchMap, h.roomId)
				}
			}
		case message := <-h.broadcast:
			//log.Println("消息" + string(message))
			for client := range h.clients {
				select {
				case client.send <- message:
					client.conn.WriteMessage(1, message)
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
