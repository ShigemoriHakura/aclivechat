package main

import (
	"flag"
	"net/http"

	"github.com/orzogc/acfundanmu"
)

const defaultAvatar = "https://tx-free-imgs.acfun.cn/style/image/defaultAvatar.jpg"

var Version = "0.1.8"
var EnableTranslate = false
var LoveText = "点亮爱心"
var FollowText = "关注了主播"
var JoinText = "加入直播间"
var QuitText = "离开直播间"
var AvatarRefreshRate = 86400
var BanString []string
var UserMarks = make(map[string]string)
var ACConnMap = make(map[int](*Hub))
var ACWatchMap = make(map[int]*[]acfundanmu.WatchingUser)
var ACPhotoMap = make(map[int64]*PhotoStruct)

var ACUsername = flag.String("username", "", "ACFun login phone/email")
var ACPassword = flag.String("password", "", "ACFun login password")
var ACCookies []*http.Cookie
