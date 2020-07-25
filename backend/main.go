package main 

import(
    "log"
    "fmt"
    "time"
    "regexp"
    "strings"
    "strconv"
    "context"
    "net/http"
    "io/ioutil"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/orzogc/acfundanmu"
    "github.com/json-iterator/go"
    "github.com/akkuman/parseConfig"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type dataGift struct {
	Id          int64     `json:"id"`// 用户ID
	AvatarUrl   string    `json:"avatarUrl"`// 礼物URL
	Timestamp   int64     `json:"timestamp"`// 发送时间
	AuthorName  string    `json:"authorName"`// 用户名
	GiftName    string    `json:"giftName"`// 礼物的描述
	Num         int       `json:"num"`// 礼物的数量
	TotalCoin   int       `json:"totalCoin"`// 礼物价格，非免费礼物时单位为AC币，免费礼物（香蕉）时为1
}

type dataUser struct {
	Id          int64     `json:"id"`// 用户ID
	AvatarUrl   string    `json:"avatarUrl"`// 头像URL
	Timestamp   int64     `json:"timestamp"`// 发送时间
    AuthorName  string    `json:"authorName"`// 用户名
    AuthorType    int     `json:"authorType"`
    PrivilegeType int     `json:"privilegeType"`
    Translation   string  `json:"translation"`
    Content       string  `json:"content"`
}

type dataGiftStruct struct {
    Cmd        int  `json:"cmd"`
    Data       dataGift `json:"data"`
}

type dataUserStruct struct {
    Cmd        int  `json:"cmd"`
    Data       dataUser `json:"data"`
}

var BanString []string
var AConnMap = make(map[int](*Hub))
var PhotoMap = make(map[int64]string)

func getACUserPhoto(id int64) (string, error){
    client := &http.Client{}
    var str =  strconv.Itoa(int(id))
    var url = "https://www.acfun.cn/u/" + str
    req, err := http.NewRequest("GET", url, nil)

    if err != nil {
        log.Fatalln(err)
        return "", err
    }

    req.Header.Set("User-Agent", "Chrome/83.0.4103.61")

    resp, err := client.Do(req)
    if err != nil {
        log.Fatalln(err)
        return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if(err != nil){
        return "", err
    }

    var cleanBody = strings.Replace(string(body), " ", "", -1)
    cleanBody = strings.Replace(cleanBody, "\n", "", -1)
    var hrefRegexp = regexp.MustCompile("(?m)ac-space-info.cover.user-photo{background:url\\(.*\\)0%0%/100%no-repeat;\\}")
    match := hrefRegexp.FindStringSubmatch(cleanBody)
    if(match != nil){
        var matches = match[0]
        matches = strings.Replace(matches, "ac-space-info.cover.user-photo{background:url(", "", -1)
        matches = strings.Replace(matches, ")0%0%/100%no-repeat;}", "", -1)
        log.Printf("UserId(%v) match: %v", str, matches)
        return matches, nil
    }
    return "", nil
}

func serveHome(w http.ResponseWriter, r *http.Request) {
    var conn, err = upgrader.Upgrade(w, r, nil)
    if(err != nil){
        log.Println("Serve: ", err)
    }else{
        log.Println("New Conn: ", fmt.Sprintf("%s", conn.RemoteAddr().String()))
        go serveWS(conn)
    }
}

func serveWS(conn *websocket.Conn){
    for{
        _, msg, err := conn.ReadMessage()
        if(err != nil){
            log.Println("Conn Err: ", err) 
            conn.Close()
            break
        }else{
            //log.Println("Conn: ", mType, string(msg))
            any := jsoniter.Get(msg)
            var cmd = any.Get("cmd").ToString()
            //log.Println("Conn cmd: ", cmd)
            switch (cmd){
                case "0":
                    conn.WriteMessage(1, []byte(`{}`))
                    break
                case "1":
                    var roomID = any.Get("data", "roomId").ToInt()
                    log.Println("Conn roomID: ", roomID)
                    if _, ok := AConnMap[roomID]; !ok {
                        AConnMap[roomID] = newHub()
                        AConnMap[roomID].htype = 1
                        AConnMap[roomID].roomId = roomID
                        go AConnMap[roomID].run()
                        go startACWS(AConnMap[roomID], roomID)
                    }
                    client := &Client{hub: AConnMap[roomID], conn: conn, send: make(chan []byte, 8192)}
                    client.hub.register <- client
                    go client.readPump()
                    return
            }
        }
    }
}

func startACWS(hub *Hub, roomID int){
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    // uid为主播的uid
    dq := acfundanmu.Start(ctx, roomID)
    var hubTime = hub.timeStamp
    for {
        if hhub, ok := AConnMap[roomID]; !ok {
            log.Println("无用户请求，关闭直播间监听", roomID)
            //cancel()
            break
            //return
        }else{
            if(hubTime != hhub.timeStamp){
                log.Println("时间戳不匹配，关闭", roomID)
                break
            }
        }
        json := jsoniter.ConfigCompatibleWithStandardLibrary
        if danmu := dq.GetDanmu(); danmu != nil {
            for _, d := range danmu {
                var val = []byte(`{}`)
                avatar, ok := PhotoMap[d.UserID]
                if !ok {
                    aavatar, err := getACUserPhoto(d.UserID)
                    if(err != nil){
                        avatar = ""
                    }
                    if(aavatar != ""){
                        PhotoMap[d.UserID] = aavatar
                    }
                    avatar = aavatar
                }else{
                    avatar = PhotoMap[d.UserID] 
                }
                //avatar = PhotoMap[d.UserID] 
                //log.Println("Data Photo", avatar)
                // 根据Type处理弹幕
                switch d.Type {
                case acfundanmu.Comment:
                    if(!checkComments(d.Comment)){
                        var data = new(dataUserStruct)
                        data.Cmd = 1
                        data.Data.Id = d.UserID
                        data.Data.AvatarUrl = avatar
                        data.Data.Timestamp = time.Now().Unix()
                        data.Data.AuthorName = d.Nickname
                        data.Data.AuthorType = 0
                        data.Data.PrivilegeType = 0
                        data.Data.Content = d.Comment
                        ddata, err := json.Marshal(data)
                        if(err == nil){
                            val = ddata
                            //log.Println("Conn Comment", string(ddata))
                        }
                    }
                    log.Printf("%s（%d）：%s\n", d.Nickname, d.UserID, d.Comment)
                case acfundanmu.Like:
                    log.Printf("%s（%d）点赞\n", d.Nickname, d.UserID)
                case acfundanmu.EnterRoom:
                    var data = new(dataUserStruct)
                    data.Cmd = 1
                    data.Data.Id = d.UserID
                    data.Data.AvatarUrl = avatar
                    data.Data.Timestamp = time.Now().Unix()
                    data.Data.AuthorName = d.Nickname
                    data.Data.AuthorType = 0
                    data.Data.PrivilegeType = 0
                    data.Data.Content = "加入直播间"
                    ddata, err := json.Marshal(data)
                    if(err == nil){
                        val = ddata
                        //log.Println("Conn Join", string(ddata))
                    }
                    log.Printf("%s（%d）进入直播间\n", d.Nickname, d.UserID)
                case acfundanmu.FollowAuthor:
                    log.Printf("%s（%d）关注了主播\n", d.Nickname, d.UserID)
                case acfundanmu.ThrowBanana:
                    log.Printf("%s（%d）送出香蕉 * %d\n", d.Nickname, d.UserID, d.BananaCount)
                case acfundanmu.Gift:
                    var data = new(dataGiftStruct)
                    data.Cmd = 3
                    data.Data.Id = d.UserID
                    data.Data.AvatarUrl = d.Gift.WebpPic
                    data.Data.Timestamp = time.Now().Unix()
                    data.Data.AuthorName = d.Nickname
                    data.Data.GiftName = d.Gift.Name
                    data.Data.Num = d.Gift.Count
                    var price = d.Gift.Price * 100
                    if(d.Gift.Name == "香蕉"){
                        price = 0
                    }
                    data.Data.TotalCoin = price
                    ddata, err := json.Marshal(data)
                    if(err == nil){
                        val = ddata
                        //log.Println("Conn Gift", string(ddata))
                    }
                    //log.Println("Conn Gift", data)
                    log.Printf("%s（%d）送出礼物 %s * %d，连击数：%d\n", d.Nickname, d.UserID, d.Gift.Name, d.Gift.Count, d.Gift.Combo)
                }

                hub.broadcast <- val
            }
        } else {
            log.Println("直播结束")
            time.Sleep(5 * time.Second)
            go startACWS(AConnMap[roomID], roomID)
            break
            //return
        }
    }
}

func (c *Client) readPump() {
	defer func() {
        log.Println("用户结束")
		c.hub.unregister <- c
		c.conn.Close()
	}()
	//c.conn.SetReadLimit(512)
	//c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	//c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		//message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		//c.hub.broadcast <- message
	}
}

func checkComments(comment string)(bool){
    for _, word := range BanString {
		if(strings.Contains(comment,word)){
            return true
        }
    }
    return false
}

type Hub struct {
    htype int
    roomId int
    timeStamp int64
	clients map[*Client]bool
	broadcast chan []byte
	register chan *Client
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
        htype:   1,
        roomId: -1,
        timeStamp: time.Now().Unix(),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

type Client struct {
	hub *Hub
	conn *websocket.Conn
	send chan []byte
}

func (h *Hub) run() {
    //var ii = 0
	for {
		select {
		case client := <-h.register:
            h.clients[client] = true
            log.Println("新用户")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
                log.Println("用户断开")
                if(len(h.clients) <= 0){
                    log.Println("用户为0，关闭直播间监听", h.roomId)
                    delete(AConnMap, h.roomId)
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


func main(){
    var config = parseConfig.New("config.json")
    var BanWords = config.Get("BanWords").([]interface{})
    for _,v := range BanWords {
        BanString = append(BanString, v.(string))
    }

    log.Println("启动中，ACLiveChat，0.0.8")

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
        w.Write([]byte(`{"version": "v0.0.8", "config": {"enableTranslate": false}}`))
    })
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))
    http.Handle("/", r)
    err := http.ListenAndServe("0.0.0.0:12451", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}

func checkACConn(){
    /*var newConn = make(map[string]([]websocket.Conn))
    for k, v := range AConnMap {
        var newConnWS = []websocket.Conn{}
        log.Println("处理AC房间：" + k)
        if(len(v) > 0){
            for _, vv := range v{
                //if(vv.stauts){
                    newConnWS = append(newConnWS, vv)
                //}
            }
            if(len(newConnWS) > 0){
                newConn[k] = newConnWS
            }
        }
    }
    AConnMap = newConn*/
}

func indexOfString(element string, data []string) (int) {
    for k, v := range data {
        if element == v {
            return k
        }
    }
    return -1    //not found.
}