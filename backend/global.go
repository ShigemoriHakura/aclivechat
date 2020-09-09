package main

import (
	"flag"
	"net/http"
	"github.com/orzogc/acfundanmu"
)

var Version = "0.1.6"
var HideGift = false
var EnableTranslate = false
var NormalGift = "一般"
var YAAAAAGift = "高端"
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