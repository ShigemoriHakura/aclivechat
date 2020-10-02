package main

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func startHttpServer() {
	r := mux.NewRouter()
	r.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		serveHome(w, r)
	})
	r.HandleFunc("/room/{key}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/index.html")
	})
	r.HandleFunc("/stylegen", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/index.html")
	})
	r.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/index.html")
	})
	r.HandleFunc("/server_info", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"version": "` + Version + `", "config": {"enableTranslate": ` + strconv.FormatBool(EnableTranslate) + `}}`))
	})
	r.HandleFunc("/room_info", func(w http.ResponseWriter, r *http.Request) {
		var roomStr = ""
		ACConnMap.Lock()
		for _, v := range ACConnMap.hubMap {
			roomStr += strconv.Itoa(v.roomId) + ","
		}
		ACConnMap.Unlock()
		roomStr = trimLastChar(roomStr)
		w.Write([]byte(`{"version": "` + Version + `", "rooms": "` + roomStr + `"}`))
	})
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))
	http.Handle("/", r)

	log.Println("[Main]", "等待用户连接")
	err := http.ListenAndServe("0.0.0.0:12451", nil)
	if err != nil {
		log.Println("[Main]", "发生主端口监听错误: ", err)
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	var conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS Home]", "发生处理错误: ", err)
	} else {
		log.Println("[WS Home]", "新的前端WS连接：", fmt.Sprintf("%s", conn.RemoteAddr().String()))
		go serveWS(conn)
	}
}

func serveWS(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("[WS Server]", "发生连接错误: ", err)
			conn.Close()
			break
		} else {
			//log.Println("Conn: ", mType, string(msg))
			any := jsoniter.Get(msg)
			var cmd = any.Get("cmd").ToString()
			//log.Println("Conn cmd: ", cmd)
			switch cmd {
			case "0":
				conn.WriteMessage(1, []byte(`{}`))
				break
			case "1":
				var roomID = any.Get("data", "roomId").ToInt()
				log.Println("[WS Server]", "请求房间ID：", roomID)
				ACConnMap.Lock()
				ConnM, ok := ACConnMap.hubMap[roomID]
				ACConnMap.Unlock()
				if !ok {
					var data = new(Message)
					data.RoomID = roomID
					RoomQ.Enqueue(data)
					ACConnMap.Lock()
					ACConnMap.hubMap[roomID] = newHub()
					ACConnMap.hubMap[roomID].roomId = roomID
					go ACConnMap.hubMap[roomID].run()
					ACConnMap.Unlock()
					conn.Close()
					return
				}
				client := &Client{hub: ConnM, conn: conn, send: make(chan []byte, 8192)}
				client.hub.register <- client
				go client.readPump()
				var data = new(dataUserStruct)
				data.Cmd = 2
				data.Data.Id = 0
				data.Data.AvatarUrl = "https://tx-free-imgs.acfun.cn/style/image/defaultAvatar.jpg"
				data.Data.Timestamp = time.Now().Unix()
				data.Data.AuthorName = "弹幕姬"
				data.Data.AuthorType = 0
				data.Data.PrivilegeType = 0
				data.Data.Content = "连接成功~"
				data.Data.UserMark = ""
				//data.Data.Medal = d.Medal
				json := jsoniter.ConfigCompatibleWithStandardLibrary
				ddata, err := json.Marshal(data)
				if err == nil {
					conn.WriteMessage(1, ddata)
				}
				return
			}
		}
	}
}
