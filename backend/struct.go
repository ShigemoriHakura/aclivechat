package main

import (
	"github.com/orzogc/acfundanmu"
	"github.com/gorilla/websocket"
)

type dataGift struct {
	Id         int64  `json:"id"`         // 用户ID
	AvatarUrl  string `json:"avatarUrl"`  // 礼物URL
	Timestamp  int64  `json:"timestamp"`  // 发送时间
	AuthorName string `json:"authorName"` // 用户名
	GiftName   string `json:"giftName"`   // 礼物的描述
	Num        int    `json:"num"`        // 礼物的数量
	TotalCoin  int    `json:"totalCoin"`  // 礼物价格，非免费礼物时单位为AC币，免费礼物（香蕉）时为1
}

type dataUser struct {
	Id            int64  `json:"id"`         // 用户ID
	AvatarUrl     string `json:"avatarUrl"`  // 头像URL
	Timestamp     int64  `json:"timestamp"`  // 发送时间
	AuthorName    string `json:"authorName"` // 用户名
	AuthorType    int    `json:"authorType"`
	PrivilegeType int    `json:"privilegeType"`
	Translation   string `json:"translation"`
	Content       string `json:"content"`
	UserMark      string `json:"userMark"`
	Medal         acfundanmu.MedalInfo `json:"medalInfo"` // 粉丝牌
}

type dataGiftStruct struct {
	Cmd  int      `json:"cmd"`
	Data dataGift `json:"data"`
}

type dataUserStruct struct {
	Cmd  int      `json:"cmd"`
	Data dataUser `json:"data"`
}

type PhotoStruct struct {
	Url       string      `json:"url"`
	Timestamp int64       `json:"timestamp"`
}

type Hub struct {
	htype      int
	roomId     int
	timeStamp  int64
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}