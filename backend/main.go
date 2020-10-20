package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/akkuman/parseConfig"
	jsoniter "github.com/json-iterator/go"
	"github.com/orzogc/acfundanmu"
)

func main() {
	defer func() {
		log.Println("[Main]", "请按回车关闭。。。")
		for {
			consoleReader := bufio.NewReaderSize(os.Stdin, 1)
			_, _ = consoleReader.ReadByte()
			os.Exit(0)
		}
	}()

	ACConnMap.hubMap = make(map[int]*Hub)
	ACPhotoMap.photoMap = make(map[int64]*PhotoStruct)
	ACRoomMap.roomMap = make(map[int]struct{})

	flag.Parse()
	loginToACFun()
	log.Println("[Main]", "读取配置文件中")
	importConfig()
	log.Println("[Main]", "启动中，AcLiveChat，", BackendVersion)
	log.Println("[Main]", "头像缓存时间：", AvatarRefreshRate, "秒")
	startMessageQueue()
	startRoomQueue()
	go processMessageQueue()
	go processRoomQueue()
	go processRoomRetryQueue()
	startHttpServer()
}

func importConfig() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[Main]", "发生配置文件错误：", r)
			log.Println("[Main]", "跳过配置文件使用默认值")
		}
	}()

	var config = parseConfig.New("config.json")
	var BanWords = config.Get("BanWords").([]interface{})
	var UserMark = config.Get("UserMarks").(map[string]interface{})
	for _, v := range BanWords {
		BanString = append(BanString, v.(string))
	}
	for k, v := range UserMark {
		UserMarks[k] = v.(string)
	}

	LoveText = config.Get("LoveText").(string)
	FollowText = config.Get("FollowText").(string)
	JoinText = config.Get("JoinText").(string)
	JoinClubText = config.Get("JoinClubText").(string)
	QuitText = config.Get("QuitText").(string)
	AvatarRefreshRate = int(config.Get("AvatarRefreshRate").(float64))
}

func loginToACFun() {
	if *ACUsername != "" && *ACPassword != "" {
		log.Println("[Main]", "尝试登录ACFun账号中")
		cookies, err := acfundanmu.Login(*ACUsername, *ACPassword)
		if err != nil {
			log.Println("[Main]", *ACUsername, "登录出错：", err)
		} else {
			log.Println("[Main]", *ACUsername, "登录成功")
			ACCookies = cookies
		}
	}
}

func startMessageQueue() {
	MessageQ := initMessageQueue()
	log.Println("[Message Queue]", "初始化成功，当前队列长度：", MessageQ.Size())
}

func startRoomQueue() {
	RoomQ := initRoomQueue()
	log.Println("[Room Queue]", "初始化成功，当前队列长度：", RoomQ.Size())
}

func processMessageQueue() {
	for {
		for !MessageQ.IsEmpty() {
			tmp := MessageQ.Dequeue()
			log.Println("[Message Queue]", tmp.RoomID, "处理消息")
			ACConnMap.Lock()
			connHub, ok := ACConnMap.hubMap[tmp.RoomID]
			ACConnMap.Unlock()
			if ok {
				json := jsoniter.ConfigCompatibleWithStandardLibrary
				ddata, err := json.Marshal(tmp.Data)
				if err == nil {
					//log.Println("Sent: ", 1, string(ddata))
					connHub.broadcast <- ddata
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func processRoomQueue() {
	for {
		for !RoomQ.IsEmpty() {
			tmp := RoomQ.Dequeue()
			log.Println("[Room Queue]", tmp.RoomID, "处理房间")
			ACRoomMap.Lock()
			_, ok := ACRoomMap.roomMap[tmp.RoomID]
			ACRoomMap.Unlock()
			if !ok {
				log.Println("[Room Queue]", tmp.RoomID, "建立WS链接")
				go startACWS(tmp.RoomID)
			} else {
				log.Println("[Room Queue]", tmp.RoomID, "已存在，不新建")
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func processRoomRetryQueue() {
	for {
		time.Sleep(10 * time.Second)
		//log.Println("[Room Retry Queue]", "检查存在Hub但是不存在弹幕服务的房间")
		ACConnMap.Lock()
		for _, v := range ACConnMap.hubMap {
			//log.Println("[Room Retry Queue]", "检查", v.roomId)
			ACRoomMap.Lock()
			if _, ok := ACRoomMap.roomMap[v.roomId]; !ok {
				log.Println("[Room Retry Queue]", v.roomId, "建立WS链接")
				ACRoomMap.roomMap[v.roomId] = struct{}{}
				go startACWS(v.roomId)
			}
			ACRoomMap.Unlock()
		}
		ACConnMap.Unlock()
		//log.Println("[Room Retry Queue]", "检查完成")
	}
}

func startACWS(roomID int) {
	ACRoomMap.Lock()
	ACRoomMap.roomMap[roomID] = struct{}{}
	ACRoomMap.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		log.Println("[Danmaku]", roomID, "结束")
		ACRoomMap.Lock()
		delete(ACRoomMap.roomMap, roomID)
		ACRoomMap.Unlock()
		cancel()
	}()
	log.Println("[Danmaku]", roomID, "WS监听服务启动中")
	// uid为主播的uid
	dq, err := acfundanmu.Init(int64(roomID), ACCookies)
	if err != nil {
		log.Println("[Danmaku]", roomID, "出错结束")
		return
	}
	dq.StartDanmu(ctx, false)
	go func() {
		var watchingListold []acfundanmu.WatchingUser
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

							var dataQ = new(Message)
							dataQ.RoomID = roomID
							dataQ.Data = data
							MessageQ.Enqueue(dataQ)
							log.Printf("[Danmaku] %v, %s（%d）离开直播间\n", roomID, d.Nickname, d.UserID)
						}
					}
				}
				watchingListold = watchingList
				time.Sleep(5 * time.Second)
			}
		}
	}()
	for {
		if danmu := dq.GetDanmu(); danmu != nil {
			for _, d := range danmu {
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
						var dataQ = new(Message)
						dataQ.RoomID = roomID
						dataQ.Data = data
						MessageQ.Enqueue(dataQ)
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
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
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
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
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
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
					log.Printf("[Danmaku] %v, %s（%d）关注了主播\n", roomID, d.Nickname, d.UserID)
				case *acfundanmu.ThrowBanana:
					var data = new(dataGiftStruct)
					data.Cmd = 3
					data.Data.Id = d.UserID
					data.Data.AvatarUrl = avatar
					data.Data.WebpPic = "https://static.yximgs.com/bs2/giftCenter/giftCenter-20200316101317UbXssBoH.webp"
					data.Data.PngPic = "https://static.yximgs.com/bs2/giftCenter/giftCenter-20200812141711JRxMyUWH.png"
					data.Data.Timestamp = time.Now().Unix()
					data.Data.AuthorName = d.Nickname
					data.Data.UserMark = getUserMark(d.UserID)
					data.Data.Medal = d.Medal
					data.Data.GiftName = "香蕉"
					data.Data.Num = d.BananaCount
					data.Data.TotalCoin = 0
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
					log.Printf("[Danmaku] %v, %s（%d）送出香蕉 * %d\n", roomID, d.Nickname, d.UserID, d.BananaCount)
				case *acfundanmu.Gift:
					var data = new(dataGiftStruct)
					data.Cmd = 3
					data.Data.Id = d.UserID
					data.Data.AvatarUrl = avatar
					data.Data.WebpPic = d.WebpPic
					data.Data.PngPic = d.PngPic
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
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
					//log.Println("Conn Gift", data)
					log.Printf("[Danmaku] %v, %s（%d）送出礼物 %s * %d，连击数：%d\n", roomID, d.Nickname, d.UserID, d.GiftName, d.Count, d.Combo)
				case *acfundanmu.JoinClub:
					var data = new(dataUserStruct)
					data.Cmd = 11
					data.Data.Id = d.FansInfo.UserID
					data.Data.AvatarUrl = avatar
					data.Data.Timestamp = time.Now().Unix()
					data.Data.AuthorName = d.FansInfo.Nickname
					data.Data.AuthorType = AuthorType
					data.Data.PrivilegeType = 0
					data.Data.Content = JoinClubText
					data.Data.UserMark = getUserMark(d.FansInfo.UserID)
					var dataQ = new(Message)
					dataQ.RoomID = roomID
					dataQ.Data = data
					MessageQ.Enqueue(dataQ)
					log.Printf("%s（%d）加入主播%s（%d）的守护团", d.FansInfo.Nickname, d.FansInfo.UserID, d.UperInfo.Nickname, d.UperInfo.UserID)
				}
			}
		} else {
			log.Println("[Danmaku]", roomID, " 直播结束")
			return
		}
	}
}
