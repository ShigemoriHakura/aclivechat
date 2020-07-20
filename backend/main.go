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

var ConnMap = make(map[string]([]websocket.Conn))
var PhotoMap = make(map[int64]string)

func getUserPhoto(id int64) (string, error){
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
        go func(conn *websocket.Conn){
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
                            //log.Println("Conn roomID: ", roomID)
                            go func(conn *websocket.Conn, roomID int){
                                ctx, cancel := context.WithCancel(context.Background())
                                defer cancel()
                                // uid为主播的uid
                                dq := acfundanmu.Start(ctx, roomID)
                                for {
                                    json := jsoniter.ConfigCompatibleWithStandardLibrary
                                    if danmu := dq.GetDanmu(); danmu != nil {
                                        for _, d := range danmu {
                                            var val = []byte(`{}`)
                                            var avatar = ""
                                            //avatar, err = getUserPhoto(d.UserID)
                                            if _, ok := PhotoMap[d.UserID]; !ok {
                                                avatar, err = getUserPhoto(d.UserID)
                                                if(err != nil){
                                                    avatar = ""
                                                }
                                                if(avatar != ""){
                                                    PhotoMap[d.UserID] = avatar
                                                }
                                            }else{
                                                avatar = PhotoMap[d.UserID] 
                                            }
                                            //log.Println("Data Photo", avatar)
                                            // 根据Type处理弹幕
                                            switch d.Type {
                                            case acfundanmu.Comment:
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
                                            
                                            var err = conn.WriteMessage(1, val)
                                            if(err != nil){
                                                log.Println("错误，可能链接断开，我懒得处理了")
                                                conn.Close()
                                                return
                                            }
                                        }
                                    } else {
                                        log.Println("直播结束")
                                        conn.Close()
                                        break
                                    }
                                }
                            }(conn, roomID)
                            break
                    }
                }
            }
        }(conn)
    }
}

func main(){
    log.Println("启动中，ACLiveChat，0.0.6")
    r := mux.NewRouter()
    r.HandleFunc("/chat", serveHome)
    r.HandleFunc("/room/{key}", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "frontend/dist/index.html")
    })
    r.HandleFunc("/stylegen", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "frontend/dist/index.html")
    })
    r.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "frontend/dist/index.html")
    })
    r.HandleFunc("/server_info", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"version": "v0.0.4", "config": {"enableTranslate": false}}`))
    })
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("frontend/dist")))
    http.Handle("/", r)
    err := http.ListenAndServe("0.0.0.0:12451", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}