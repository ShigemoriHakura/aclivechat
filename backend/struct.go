package main

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/orzogc/acfundanmu"
)

type dataGift struct {
	Id         int64                `json:"id"`         // 用户ID
	AvatarUrl  string               `json:"avatarUrl"`  // 用户头像URL
	WebpPic    string               `json:"webpPicUrl"` // 礼物的webp格式图片（动图）
	PngPic     string               `json:"pngPicUrl"`  // 礼物的png格式图片（大）
	Timestamp  int64                `json:"timestamp"`  // 发送时间
	AuthorName string               `json:"authorName"` // 用户名
	AuthorType int                  `json:"authorType"` // 房管类型
	UserMark   string               `json:"userMark"`
	Medal      acfundanmu.MedalInfo `json:"medalInfo"` // 粉丝牌
	GiftName   string               `json:"giftName"`  // 礼物的描述
	Num        int                  `json:"num"`       // 礼物的数量
	TotalCoin  int                  `json:"totalCoin"` // 礼物价格，非免费礼物时单位为AC币，免费礼物（香蕉）时为1
}

type dataUser struct {
	Id            int64                `json:"id"`         // 用户ID
	AvatarUrl     string               `json:"avatarUrl"`  // 头像URL
	Timestamp     int64                `json:"timestamp"`  // 发送时间
	AuthorName    string               `json:"authorName"` // 用户名
	AuthorType    int                  `json:"authorType"` // 房管类型
	PrivilegeType int                  `json:"privilegeType"`
	Translation   string               `json:"translation"`
	Content       string               `json:"content"`
	UserMark      string               `json:"userMark"`
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
	Url       string `json:"url"`
	Timestamp int64  `json:"timestamp"`
}

//处理好的信息
type Message struct {
	RoomID int
	Data   interface{}
}

//信息队列
type MessageQueue struct {
	sync.Mutex
	Messages []*Message
}

type IMessageQueue interface {
	New() MessageQueue
	Enqueue(t Message)
	Dequeue(t Message)
	IsEmpty() bool
	Size() int
}

// 入队
func (q *MessageQueue) Enqueue(data *Message) {
	q.Lock()
	defer q.Unlock()
	q.Messages = append(q.Messages, data)
}

// 出队
func (q *MessageQueue) Dequeue() *Message {
	q.Lock()
	defer q.Unlock()
	Message := q.Messages[0]
	q.Messages = q.Messages[1:len(q.Messages)]
	return Message
}

// 队列是否为空
func (q *MessageQueue) IsEmpty() bool {
	q.Lock()
	defer q.Unlock()
	return len(q.Messages) == 0
}

// 队列长度
func (q *MessageQueue) Size() int {
	q.Lock()
	defer q.Unlock()
	return len(q.Messages)
}

func initMessageQueue() *MessageQueue {
	if MessageQ.Messages == nil {
		MessageQ = MessageQueue{}
	}
	return &MessageQ
}

func initRoomQueue() *MessageQueue {
	if RoomQ.Messages == nil {
		RoomQ = MessageQueue{}
	}
	return &RoomQ
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
