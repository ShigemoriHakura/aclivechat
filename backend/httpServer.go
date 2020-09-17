package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/orzogc/acfundanmu"
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
				if _, ok := ACConnMap[roomID]; !ok {
					ACConnMap[roomID] = newHub()
					ACConnMap[roomID].htype = 1
					ACConnMap[roomID].roomId = roomID
					go ACConnMap[roomID].run()
					go startACWS(ACConnMap[roomID], roomID)
				}
				client := &Client{hub: ACConnMap[roomID], conn: conn, send: make(chan []byte, 8192)}
				client.hub.register <- client
				go client.readPump()
				return
			}
		}
	}
}

func startACWS(hub *Hub, roomID int) {
	if hub == nil {
		log.Println("[Danmaku]", roomID, "这不合理")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		log.Println("[Danmaku]", roomID, "结束")
		cancel()
	}()
	log.Println("[Danmaku]", roomID, "WS监听服务启动中")
	// uid为主播的uid
	dq, err := acfundanmu.Init(int64(roomID), ACCookies)
	if err != nil {
		//log.Println(err)
		log.Println("[Danmaku]", roomID, "5秒后重试")
		time.Sleep(5 * time.Second)
		log.Println("[Danmaku]", roomID, "重试启动")
		if ACConnMap[roomID] != nil {
			go startACWS(ACConnMap[roomID], roomID)
		} else {
			log.Println("[Danmaku]", roomID, "没监听了，关！")
		}
		return
	}
	dq.StartDanmu(ctx)
	if hub != nil {
		var hubTime = hub.timeStamp
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					// 循环获取watchingList并处理
					watchingList, err := dq.GetWatchingList()
					if err != nil {
						log.Println("[Danmaku]", roomID, "获取在线用户失败：", err)
					} else {
						watchingListold, ok := ACWatchMap[roomID]
						if !ok {
							ACWatchMap[roomID] = watchingList
							//return
						} else {
							ACWatchMap[roomID] = watchingList

							//处理旧的
							var processedList []string
							processedList2 := make(map[string]acfundanmu.WatchingUser)
							for _, value := range watchingListold {
								var stringUserID = strconv.FormatInt(value.UserID, 10)
								processedList = append(processedList, stringUserID)
								processedList2[stringUserID] = value
							}

							//处理新的
							var processedNewList []string
							for _, value := range watchingList {
								var stringUserID = strconv.FormatInt(value.UserID, 10)
								//fmt.Printf("id %v \n", stringUserID)
								processedNewList = append(processedNewList, stringUserID)
							}
							_, removed := Arrcmp(processedList, processedNewList)
							for _, value := range removed {
								d := processedList2[value]
								if !d.AnonymousUser {
									var val = []byte(`{}`)
									avatar, AuthorType := getAvatarAndAuthorType(d.UserInfo, roomID)
									var data = new(dataUserStruct)
									data.Cmd = 9
									data.Data.Id = d.UserID
									data.Data.AvatarUrl = avatar
									data.Data.Timestamp = time.Now().Unix()
									data.Data.AuthorName = d.Nickname
									data.Data.AuthorType = AuthorType
									data.Data.PrivilegeType = 0
									data.Data.Content = QuitText
									data.Data.UserMark = getUserMark(d.UserID)
									//data.Data.Medal = d.Medal
									json := jsoniter.ConfigCompatibleWithStandardLibrary
									ddata, err := json.Marshal(data)
									if err == nil {
										val = ddata
										//log.Println("Conn Join", string(ddata))
									}
									hub.broadcast <- val
									log.Printf("[Danmaku] %v, %s（%d）离开直播间\n", roomID, d.Nickname, d.UserID)
									//fmt.Printf("id %v \n", value)
								}
							}
							//fmt.Printf("add: %v rem: %v old: %v new: %v \n", added, removed, processedList, processedNewList)
						}
					}
					time.Sleep(5 * time.Second)
				}
			}
		}()
		for {
			if hhub, ok := ACConnMap[roomID]; !ok {
				log.Println("[Danmaku]", roomID, "无用户请求，关闭直播间监听")
				//cancel()
				break
				//return
			} else {
				if hubTime != hhub.timeStamp {
					log.Println("[Danmaku]", roomID, "时间戳不匹配，关闭")
					break
				}
			}
			json := jsoniter.ConfigCompatibleWithStandardLibrary
			if danmu := dq.GetDanmu(); danmu != nil {
				for _, d := range danmu {
					var val = []byte(`{}`)
					avatar, AuthorType := getAvatarAndAuthorType(d.GetUserInfo(), roomID)
					// 根据Type处理弹幕
					switch d := d.(type) {
					case *acfundanmu.Comment:
						if !checkComments(d.Content) {
							var data = new(dataUserStruct)
							data.Cmd = 2
							data.Data.Id = d.UserID
							data.Data.AvatarUrl = avatar
							data.Data.Timestamp = time.Now().Unix()
							data.Data.AuthorName = d.Nickname
							data.Data.AuthorType = AuthorType
							data.Data.PrivilegeType = 0
							data.Data.Content = d.Content
							data.Data.UserMark = getUserMark(d.UserID)
							data.Data.Medal = d.Medal
							ddata, err := json.Marshal(data)
							if err == nil {
								val = ddata
								//log.Println("Conn Comment", string(ddata))
							}
						}
						log.Printf("[Danmaku] %v, %s（%d）：%s\n", roomID, d.Nickname, d.UserID, d.Content)
					case *acfundanmu.Like:
						var data = new(dataUserStruct)
						data.Cmd = 8
						data.Data.Id = d.UserID
						data.Data.AvatarUrl = avatar
						data.Data.Timestamp = time.Now().Unix()
						data.Data.AuthorName = d.Nickname
						data.Data.AuthorType = AuthorType
						data.Data.PrivilegeType = 0
						data.Data.Content = LoveText
						data.Data.UserMark = getUserMark(d.UserID)
						data.Data.Medal = d.Medal
						ddata, err := json.Marshal(data)
						if err == nil {
							val = ddata
							//log.Println("Conn Comment", string(ddata))
						}
						log.Printf("[Danmaku] %v, %s（%d）点赞\n", roomID, d.Nickname, d.UserID)
					case *acfundanmu.EnterRoom:
						var data = new(dataUserStruct)
						data.Cmd = 1
						data.Data.Id = d.UserID
						data.Data.AvatarUrl = avatar
						data.Data.Timestamp = time.Now().Unix()
						data.Data.AuthorName = d.Nickname
						data.Data.AuthorType = AuthorType
						data.Data.PrivilegeType = 0
						data.Data.Content = JoinText
						data.Data.UserMark = getUserMark(d.UserID)
						data.Data.Medal = d.Medal
						ddata, err := json.Marshal(data)
						if err == nil {
							val = ddata
							//log.Println("Conn Join", string(ddata))
						}
						log.Printf("[Danmaku] %v, %s（%d）进入直播间\n", roomID, d.Nickname, d.UserID)
					case *acfundanmu.FollowAuthor:
						var data = new(dataUserStruct)
						data.Cmd = 10
						data.Data.Id = d.UserID
						data.Data.AvatarUrl = avatar
						data.Data.Timestamp = time.Now().Unix()
						data.Data.AuthorName = d.Nickname
						data.Data.AuthorType = AuthorType
						data.Data.PrivilegeType = 0
						data.Data.Content = FollowText
						data.Data.UserMark = getUserMark(d.UserID)
						data.Data.Medal = d.Medal
						ddata, err := json.Marshal(data)
						if err == nil {
							val = ddata
							//log.Println("Conn Join", string(ddata))
						}
						log.Printf("[Danmaku] %v, %s（%d）关注了主播\n", roomID, d.Nickname, d.UserID)
					case *acfundanmu.ThrowBanana:
						var data = new(dataGiftStruct)
						data.Cmd = 3
						data.Data.Id = d.UserID
						data.Data.AvatarUrl = avatar
						data.Data.Timestamp = time.Now().Unix()
						data.Data.AuthorName = d.Nickname
						data.Data.UserMark = getUserMark(d.UserID)
						data.Data.Medal = d.Medal
						data.Data.GiftName = "香蕉"
						data.Data.Num = d.BananaCount
						data.Data.TotalCoin = 0
						ddata, err := json.Marshal(data)
						if err == nil {
							val = ddata
							//log.Println("Conn Gift", string(ddata))
						}
						log.Printf("[Danmaku] %v, %s（%d）送出香蕉 * %d\n", roomID, d.Nickname, d.UserID, d.BananaCount)
					case *acfundanmu.Gift:
						var data = new(dataGiftStruct)
						data.Cmd = 3
						data.Data.Id = d.UserID
						data.Data.AvatarUrl = avatar
						data.Data.Timestamp = time.Now().Unix()
						data.Data.AuthorName = d.Nickname
						data.Data.AuthorType = AuthorType
						data.Data.UserMark = getUserMark(d.UserID)
						data.Data.Medal = d.Medal
						data.Data.GiftName = d.GiftName
						data.Data.Num = int(d.Count)
						var price = d.Value / 10
						if d.GiftName == "香蕉" {
							price = 0
						}
						data.Data.TotalCoin = int(price)
						ddata, err := json.Marshal(data)
						if err == nil {
							val = ddata
							//log.Println("Conn Gift", string(ddata))
						}
						//log.Println("Conn Gift", data)
						log.Printf("[Danmaku] %v, %s（%d）送出礼物 %s * %d，连击数：%d\n", roomID, d.Nickname, d.UserID, d.GiftName, d.Count, d.Combo)
					}

					hub.broadcast <- val
				}
			} else {
				log.Println("[Danmaku]", roomID, " 直播结束")
				time.Sleep(5 * time.Second)
				go startACWS(ACConnMap[roomID], roomID)
				break
				//return
			}
		}
	} else {
		log.Println("[Danmaku]", roomID, "无Hub，直接鲨")
	}
}
