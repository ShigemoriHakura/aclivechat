package main

import (
	"flag"
	"sync"

	"github.com/orzogc/acfundanmu"
)

const defaultAvatar = "https://tx-free-imgs.acfun.cn/style/image/defaultAvatar.jpg"

var Version = "0.2.1"
var EnableTranslate = false
var LoveText = "点亮爱心"
var FollowText = "关注了主播"
var JoinText = "加入直播间"
var QuitText = "离开直播间"
var AvatarRefreshRate = 86400
var BanString []string
var UserMarks = make(map[string]string)
var ACWatchMap = make(map[int][]acfundanmu.WatchingUser)
var ACPhotoMap = make(map[int64]*PhotoStruct)

var ACConnMap struct {
	sync.Mutex
	hubMap map[int]*Hub
}

var ACRoomMap []int

var ACUsername = flag.String("username", "", "ACFun login phone/email")
var ACPassword = flag.String("password", "", "ACFun login password")
var ACCookies []string

var MessageQ MessageQueue
var RoomQ MessageQueue
