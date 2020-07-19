package main 

import(
    "log"
    "net/http"
    "fmt"
    "time"
    //"strconv"
    "context"
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

/*

id: data.id,
avatarUrl: data.avatarUrl,
time: new Date(data.timestamp * 1000),
authorName: data.authorName,
price: price,
giftName: data.giftName,
num: data.num
data.totalCoin
*/
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
                                            // 根据Type处理弹幕
                                            switch d.Type {
                                            case acfundanmu.Comment:
                                                var data = new(dataUserStruct)
                                                data.Cmd = 1
                                                data.Data.Id = d.UserID
                                                data.Data.AvatarUrl = "//cdn.aixifan.com/dotnet/20130418/umeditor/dialogs/emotion/images/ac/23.gif"
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = d.Nickname
                                                data.Data.AuthorType = 1
                                                data.Data.PrivilegeType = 0
                                                data.Data.Content = d.Comment
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Comment", string(ddata))
                                                }
                                                fmt.Printf("%s（%d）：%s\n", d.Nickname, d.UserID, d.Comment)
                                            case acfundanmu.Like:
                                                fmt.Printf("%s（%d）点赞\n", d.Nickname, d.UserID)
                                            case acfundanmu.EnterRoom:
                                                var data = new(dataUserStruct)
                                                data.Cmd = 1
                                                data.Data.Id = d.UserID
                                                data.Data.AvatarUrl = "//cdn.aixifan.com/dotnet/20130418/umeditor/dialogs/emotion/images/ac/23.gif"
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = d.Nickname
                                                data.Data.AuthorType = 1
                                                data.Data.PrivilegeType = 0
                                                data.Data.Content = "加入直播间"
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Join", string(ddata))
                                                }
                                                fmt.Printf("%s（%d）进入直播间\n", d.Nickname, d.UserID)
                                            case acfundanmu.FollowAuthor:
                                                fmt.Printf("%s（%d）关注了主播\n", d.Nickname, d.UserID)
                                            case acfundanmu.ThrowBanana:
                                                fmt.Printf("%s（%d）送出香蕉 * %d\n", d.Nickname, d.UserID, d.BananaCount)
                                            case acfundanmu.Gift:
                                                var data = new(dataGiftStruct)
                                                data.Cmd = 3
                                                data.Data.Id = d.UserID
                                                data.Data.AvatarUrl = d.Gift.WebpPic
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = d.Nickname
                                                data.Data.GiftName = d.Gift.Name
                                                data.Data.Num = d.Gift.Count
                                                data.Data.TotalCoin = d.Gift.Price * 100
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Gift", string(ddata))
                                                }
                                                //log.Println("Conn Gift", data)
                                                fmt.Printf("%s（%d）送出礼物 %s * %d，连击数：%d\n", d.Nickname, d.UserID, d.Gift.Name, d.Gift.Count, d.Gift.Combo)
                                            }
                                            
                                            var err = conn.WriteMessage(1, val)
                                            if(err != nil){
                                                fmt.Println("错误，可能链接断开，我懒得处理了")
                                                return
                                            }
                                        }
                                    } else {
                                        fmt.Println("直播结束")
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
    fmt.Println("启动中，ACLiveChat，0.0.1")
    r := mux.NewRouter()
    r.HandleFunc("/chat", serveHome)
    r.HandleFunc("/room/{key}", func(w http.ResponseWriter, r *http.Request) {
        http.ServeFile(w, r, "frontend/dist/index.html")
    })
    r.HandleFunc("/server_info", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"version": "v0.0.1", "config": {"enableTranslate": false}}`))
    })
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("frontend/dist")))
    http.Handle("/", r)
    err := http.ListenAndServe("0.0.0.0:12451", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}