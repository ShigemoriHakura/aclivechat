package main

import (
	"flag"
	"sync"
)

const defaultAvatar = "https://tx-free-imgs.acfun.cn/style/image/defaultAvatar.jpg"

var BackendVersion = "0.2.6"
var FrontendVersion = "0.2.8"
var EnableTranslate = false
var LoveText = "点亮爱心"
var FollowText = "关注了主播"
var JoinText = "加入直播间"
var JoinClubText = "加入了守护团"
var QuitText = "离开直播间"
var AvatarRefreshRate = 86400
var BanString []string
var UserMarks = make(map[string]string)

var ACConnMap struct {
	sync.Mutex
	hubMap map[int]*Hub
}

var ACPhotoMap struct {
	sync.Mutex
	photoMap map[int64]*PhotoStruct
}

var ACRoomMap struct {
	sync.Mutex
	roomMap map[int]struct{}
}

var ACUsername = flag.String("username", "", "ACFun login phone/email")
var ACPassword = flag.String("password", "", "ACFun login password")
var ACCookies []string

var MessageQ MessageQueue
var RoomQ MessageQueue
