package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	jsoniter "github.com/json-iterator/go"
	"github.com/orzogc/acfundanmu"
)

func getACUserPhoto(id int64) string {
	client := &http.Client{Timeout: 2 * time.Second}
	var str = strconv.Itoa(int(id))
	var url = "https://live.acfun.cn/rest/pc-direct/user/userInfo?userId=" + str
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		log.Println(err)
		return defaultAvatar
	}

	req.Header.Set("User-Agent", "Chrome/83.0.4103.61")

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return defaultAvatar
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return defaultAvatar
	}

	any := jsoniter.Get(body)
	var avatar = any.Get("profile", "headUrl").ToString()
	if avatar != "" {
		log.Printf("[Avatar] 用户(%v) 头像匹配: %v", str, avatar)
		return avatar
	}
	log.Printf("[Avatar] 用户(%v) 头像获取失败", str)
	return defaultAvatar
}

func Arrcmp(src []string, dest []string) ([]string, []string) {
	msrc := make(map[string]byte) //按源数组建索引
	mall := make(map[string]byte) //源+目所有元素建索引

	var set []string //交集

	//1.源数组建立map
	for _, v := range src {
		msrc[v] = 0
		mall[v] = 0
	}
	//2.目数组中，存不进去，即重复元素，所有存不进去的集合就是并集
	for _, v := range dest {
		l := len(mall)
		mall[v] = 1
		if l != len(mall) { //长度变化，即可以存
			l = len(mall)
		} else { //存不了，进并集
			set = append(set, v)
		}
	}
	//3.遍历交集，在并集中找，找到就从并集中删，删完后就是补集（即并-交=所有变化的元素）
	for _, v := range set {
		delete(mall, v)
	}
	//4.此时，mall是补集，所有元素去源中找，找到就是删除的，找不到的必定能在目数组中找到，即新加的
	var added, deleted []string
	for v := range mall {
		_, exist := msrc[v]
		if exist {
			deleted = append(deleted, v)
		} else {
			added = append(added, v)
		}
	}

	return added, deleted
}

func checkComments(comment string) bool {
	for _, word := range BanString {
		if strings.Contains(comment, word) {
			return true
		}
	}
	return false
}

func getUserMark(uid int64) string {
	uidString := strconv.FormatInt(uid, 10)
	userMark, ok := UserMarks[uidString]
	if ok {
		return userMark
	}
	return ""
}

func getAvatarAndAuthorType(userInfo acfundanmu.UserInfo, roomID int) (string, int) {
	UserID := userInfo.UserID
	ManagerType := userInfo.ManagerType
	var AuthorType = 0
	avatar := defaultAvatar
	ACPhotoMap.Lock()
	avatarStruct, ok := ACPhotoMap.photoMap[UserID]
	ACPhotoMap.Unlock()
	saveCache := false
	getNewAvater := false
	if userInfo.Avatar != "" {
		avatar = userInfo.Avatar
		if !ok || userInfo.Avatar != avatarStruct.Url {
			saveCache = true
		}
	} else {
		if ok {
			//判断缓存
			if int(time.Now().Unix()-avatarStruct.Timestamp) > AvatarRefreshRate {
				getNewAvater = true
			} else {
				avatar = avatarStruct.Url
			}
		} else {
			getNewAvater = true
		}
	}
	if getNewAvater {
		avatar = getACUserPhoto(UserID)
		saveCache = true
	}
	if saveCache {
		newAvatarStruct := new(PhotoStruct)
		newAvatarStruct.Url = avatar
		newAvatarStruct.Timestamp = time.Now().Unix()
		ACPhotoMap.Lock()
		ACPhotoMap.photoMap[UserID] = newAvatarStruct
		ACPhotoMap.Unlock()
	}
	//log.Println("Data Photo", avatar)
	if int64(roomID) == UserID {
		AuthorType = 3
	}
	if ManagerType == acfundanmu.NormalManager {
		AuthorType = 2
	}
	return avatar, AuthorType
}

func trimLastChar(s string) string {
	r, size := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return s[:len(s)-size]
}
