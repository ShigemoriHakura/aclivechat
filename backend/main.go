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
    "biliDanMu/models"
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
	TotalCoin   int       `json:"totalCoin"`// 礼物价格
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

var HideGift bool
var HideJoin bool
var NormalGift = "一般"
var YAAAAAGift = "高端"
var BanString []string

var ConnMap = make(map[string]([]websocket.Conn))
var AWSMap = make(map[string]([]websocket.Conn))
var ACPhotoMap = make(map[int64]string)
var BPhotoMap = make(map[int64]string)

func getACUserPhoto(id int64) (string, error){
    client := &http.Client{Timeout: time.Second}
    var str =  strconv.Itoa(int(id))
    var url = "https://www.acfun.cn/u/" + str
    req, err := http.NewRequest("GET", url, nil)

    if err != nil {
        log.Println(err)
        return "", err
    }

    req.Header.Set("User-Agent", "Chrome/83.0.4103.61")

    resp, err := client.Do(req)
    if err != nil {
        log.Println(err)
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
        log.Printf("AC UserId(%v) match: %v", str, matches)
        return matches, nil
    }
    return "", nil
}

func getBUserPhoto(id int64) (string, error){
    client := &http.Client{Timeout: time.Second}
    var str =  strconv.Itoa(int(id))
    var url = "https://api.bilibili.com/x/space/acc/info?mid=" + str
    req, err := http.NewRequest("GET", url, nil)

    if err != nil {
        log.Println(err)
        return "", err
    }

    req.Header.Set("User-Agent", "Chrome/83.0.4103.61")

    resp, err := client.Do(req)
    if err != nil {
        log.Println(err)
        return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if(err != nil){
        return "", err
    }

    any := jsoniter.Get(body)
    var avatar = any.Get("data", "face").ToString()
    if(avatar != ""){
        log.Printf("B UserId(%v) match: %v", str, avatar)
        return avatar, nil
    }
    return "", nil
}

func checkComments(comment string)(bool){
    for _, word := range BanString {
		if(strings.Contains(comment,word)){
            return true
        }
    }
    return false
}

//Todo: 加入弹幕连接池，避免重复连接相同直播间造成阻断
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
                            var broomID = any.Get("data", "broomId").ToUint32()
                            log.Println("Conn aroomID: ", roomID)
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
                                            if _, ok := ACPhotoMap[d.UserID]; !ok {
                                                avatar, err = getACUserPhoto(d.UserID)
                                                if(err != nil){
                                                    avatar = ""
                                                }
                                                if(avatar != ""){
                                                    ACPhotoMap[d.UserID] = avatar
                                                }
                                            }else{
                                                avatar = ACPhotoMap[d.UserID] 
                                            }
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
                                                if(!HideJoin){
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
                                                }
                                                log.Printf("%s（%d）进入直播间\n", d.Nickname, d.UserID)
                                            case acfundanmu.FollowAuthor:
                                                log.Printf("%s（%d）关注了主播\n", d.Nickname, d.UserID)
                                            case acfundanmu.ThrowBanana:
                                                var data = new(dataGiftStruct)
                                                data.Cmd = 3
                                                data.Data.Id = d.UserID
                                                //data.Data.AvatarUrl = "https://static.yximgs.com/bs2/giftCenter/giftCenter-20200316101317UbXssBoH.webp"
                                                data.Data.AvatarUrl = avatar
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = d.Nickname
                                                if(!HideGift){
                                                    data.Data.GiftName = "香蕉"
                                                }else{
                                                    data.Data.GiftName = NormalGift
                                                }
                                                data.Data.Num = d.BananaCount
                                                data.Data.TotalCoin = 0
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Gift", string(ddata))
                                                }
                                                log.Printf("%s（%d）送出香蕉 * %d\n", d.Nickname, d.UserID, d.BananaCount)
                                            case acfundanmu.Gift:
                                                var data = new(dataGiftStruct)
                                                data.Cmd = 3
                                                data.Data.Id = d.UserID
                                                //data.Data.AvatarUrl = d.Gift.WebpPic
                                                data.Data.AvatarUrl = avatar
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = d.Nickname
                                                data.Data.GiftName = d.Gift.Name
                                                data.Data.Num = d.Gift.Count
                                                var price = d.Gift.Price * 100
                                                if(d.Gift.Name == "香蕉"){
                                                    price = 0
                                                }
                                                if(HideGift){
                                                    if(price <= 0){
                                                        data.Data.GiftName = NormalGift
                                                    }else {
                                                        data.Data.GiftName = YAAAAAGift
                                                    }
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
                            //log.Println("Conn broomID: ", broomID)
                            if(broomID > 0){
                                log.Println("Conn broomID: ", broomID)
                                go func(conn *websocket.Conn, roomid uint32){
                                    json := jsoniter.ConfigCompatibleWithStandardLibrary
                                    if roomid >= 100 && roomid < 1000 {
                                        roomid,err = models.GetRealRoomID(int(roomid))
                                        if err != nil {
                                            log.Println("房间号输入错误，请退出重新输入！")
                                            return
                                        }
                                    }

                                    c, err := models.NewClient(roomid)
                                    if err != nil {
                                        fmt.Println("models.NewClient err: ", err)
                                        return
                                    }
                                    pool, err := c.Start();
                                    if err != nil {
                                        fmt.Println("c.Start err :", err)
                                        return
                                    }

                                    for {
                                        var val = []byte(`{}`)
                                        var avatar = ""
                                        select {
                                        /*case uc := <-pool.MsgUncompressed:
                                            // 目前只处理未压缩数据的关注数变化信息
                                            if cmd := json.Get([]byte(uc), "cmd").ToString(); models.CMD(cmd) == models.CMDRoomRealTimeMessageUpdate {
                                                fans := json.Get([]byte(uc), "data", "fans").ToInt()
                                                fmt.Println("当前房间关注数变动：", fans)
                                            }*/
                                        case src := <-pool.UserMsg:
                                            m := models.NewDanmu()
                                            m.GetDanmuMsg([]byte(src))
                                            if(!checkComments(m.Text)){
                                                if _, ok := BPhotoMap[int64(m.UID)]; !ok {
                                                    avatar, err = getBUserPhoto(int64(m.UID))
                                                    if(err != nil){
                                                        avatar = ""
                                                    }
                                                    if(avatar != ""){
                                                        BPhotoMap[int64(m.UID)] = avatar
                                                    }
                                                }else{
                                                    avatar = BPhotoMap[int64(m.UID)] 
                                                }

                                                //log.Println(string([]byte(src)))
                                                var data = new(dataUserStruct)
                                                data.Cmd = 1
                                                data.Data.Id = int64(m.UID)
                                                data.Data.AvatarUrl = avatar
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = m.Uname
                                                data.Data.AuthorType = 0
                                                data.Data.PrivilegeType = 0
                                                data.Data.Content = m.Text
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Comment", string(ddata))
                                                }
                                            }
                                            log.Printf("%d-%s | %d-%s: %s\n", m.MedalLevel, m.MedalName, m.Ulevel, m.Uname, m.Text)
                                        case src := <-pool.UserGift:
                                            g := models.NewGift()
                                            g.GetGiftMsg([]byte(src))
                                            var data = new(dataGiftStruct)
                                            data.Cmd = 3
                                            data.Data.Id = int64(g.UID)
                                            data.Data.AvatarUrl = g.Face
                                            data.Data.Timestamp = time.Now().Unix()
                                            data.Data.AuthorName = g.UUname
                                            data.Data.GiftName = g.GiftName
                                            data.Data.Num = int(g.Num)
                                            var price = int(g.Price)
                                            if(g.CoinType == "silver"){
                                                price = 0
                                            }
                                            if(HideGift){
                                                if(price <= 0){
                                                    data.Data.GiftName = NormalGift
                                                }else {
                                                    data.Data.GiftName = YAAAAAGift
                                                }
                                            }
                                            data.Data.TotalCoin = price
                                            ddata, err := json.Marshal(data)
                                            if(err == nil){
                                                val = ddata
                                                //log.Println("Conn Gift", string(ddata))
                                            }
                                            log.Printf("%s %s 价值 %d 的 %s\n", g.UUname, g.Action, g.Price, g.GiftName)
                                        case src := <-pool.UserEnter:
                                            //log.Println(string([]byte(src)))
                                            name := json.Get([]byte(src), "data", "uname").ToString()
                                            uid := json.Get([]byte(src), "data", "uid").ToInt64()
                                            if(!HideJoin){
                                                if _, ok := BPhotoMap[uid]; !ok {
                                                    avatar, err = getBUserPhoto(uid)
                                                    if(err != nil){
                                                        avatar = ""
                                                    }
                                                    if(avatar != ""){
                                                        BPhotoMap[uid] = avatar
                                                    }
                                                }else{
                                                    avatar = BPhotoMap[uid] 
                                                }
                                                var data = new(dataUserStruct)
                                                data.Cmd = 1
                                                data.Data.Id = uid
                                                data.Data.AvatarUrl = avatar
                                                data.Data.Timestamp = time.Now().Unix()
                                                data.Data.AuthorName = name
                                                data.Data.AuthorType = 0
                                                data.Data.PrivilegeType = 0
                                                data.Data.Content = "加入直播间"
                                                ddata, err := json.Marshal(data)
                                                if(err == nil){
                                                    val = ddata
                                                    //log.Println("Conn Join", string(ddata))
                                                }
                                            }
                                            log.Printf("欢迎VIP %s 进入直播间", name)
                                        case src := <-pool.UserGuard:
                                            log.Println(string([]byte(src)))
                                            name := json.Get([]byte(src), "data", "username").ToString()
                                            log.Printf("欢迎房管 %s 进入直播间", name)
                                        case src := <-pool.UserEntry:
                                            log.Println(string([]byte(src)))
                                            cw := json.Get([]byte(src), "data", "copy_writing").ToString()
                                            log.Printf("%s", cw)
                                        }
                                        var err = conn.WriteMessage(1, val)
                                        if(err != nil){
                                            log.Println("错误，可能链接断开，我懒得处理了")
                                            conn.Close()
                                            return
                                        }
                                    }

                                }(conn, broomID)
                            }
                            break
                    }
                }
            }
        }(conn)
    }
}

func main(){
    //flag.BoolVar(&Hide, "hide", true, "隐藏礼物名字")
    var config = parseConfig.New("config.json")
    HideGift = config.Get("HideGift").(bool)
    HideJoin = config.Get("HideJoin").(bool)
    NormalGift = config.Get("NormalGift").(string)
    YAAAAAGift = config.Get("YAAAAAGift").(string)

    var BanWords = config.Get("BanWords").([]interface{})
    for _,v := range BanWords {
        BanString = append(BanString, v.(string))
    }
    if(HideGift){
        log.Println("隐藏礼物名字！")
    }
    log.Println("启动中，AC&BLiveChat，0.0.7")
    r := mux.NewRouter()
    r.HandleFunc("/chat", serveHome)
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
        w.Write([]byte(`{"version": "v0.0.7", "config": {"enableTranslate": false}}`))
    })
    r.PathPrefix("/").Handler(http.FileServer(http.Dir("dist")))
    http.Handle("/", r)
    err := http.ListenAndServe("0.0.0.0:12451", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}