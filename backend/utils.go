package main

import (
	"log"
	"time"
	"strings"
	"strconv"
	"net/http"
	"io/ioutil"
	"github.com/orzogc/acfundanmu"
	jsoniter "github.com/json-iterator/go"
)

func getACUserPhoto(id int64) (string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	var str = strconv.Itoa(int(id))
	var url = "https://live.acfun.cn/rest/pc-direct/user/userInfo?userId=" + str
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
	if err != nil {
		return "", err
	}

	any := jsoniter.Get(body)
	var avatar = any.Get("profile", "headUrl").ToString()
	if avatar != "" {
		log.Printf("[Avatar] 用户(%v) 头像匹配: %v", str, avatar)
		return avatar, nil
	}
	log.Printf("[Avatar] 用户(%v) 头像获取失败", str)
	return "", nil
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
	for v, _ := range mall {
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

func getAvatarAndAuthorType(d acfundanmu.DanmuMessage, roomID int)(string, int){
	var UserID int64
	var ManagerType acfundanmu.ManagerType
	switch d := d.(type) {
		case *acfundanmu.Comment:
			UserID = d.UserID
			ManagerType = d.ManagerType
		case *acfundanmu.Like:
			UserID = d.UserID
			ManagerType = d.ManagerType
		case *acfundanmu.EnterRoom:
			UserID = d.UserID
			ManagerType = d.ManagerType
		case *acfundanmu.FollowAuthor:
			UserID = d.UserID
			ManagerType = d.ManagerType
		case *acfundanmu.ThrowBanana:
			UserID = d.UserID
			ManagerType = d.ManagerType
		case *acfundanmu.Gift:
			UserID = d.UserID
			ManagerType = d.ManagerType
	}
	var AuthorType = 0
	avatar := "https://tx-free-imgs.acfun.cn/style/image/defaultAvatar.jpg"
	avatarStruct, ok := ACPhotoMap[UserID]
	getNewAvater := false
	//处理用户头像结构体
	if(!ok){
		getNewAvater = true
	}else{
		//判断缓存
		if(int(time.Now().Unix() - avatarStruct.Timestamp) > AvatarRefreshRate){
			getNewAvater = true
		}else{
			avatar = avatarStruct.Url
		}
	}
	if(getNewAvater){
		newavatar, err := getACUserPhoto(UserID)
		if err == nil && newavatar != "" {
			newAvatarStruct := new(PhotoStruct)
			newAvatarStruct.Url = newavatar
			newAvatarStruct.Timestamp = time.Now().Unix()
			ACPhotoMap[UserID] = newAvatarStruct
			avatar = newavatar
			//更新头像数组和头像
		}
	}
	//log.Println("Data Photo", avatar)
	if int64(roomID) == UserID {
		AuthorType = 3
	}
	if ManagerType == 1 {
		AuthorType = 2
	}
	return avatar, AuthorType
}