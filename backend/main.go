package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/akkuman/parseConfig"
)

func main() {
	defer func() {
		log.Println("[Main]", "请按回车关闭。。。")
		for {
			consoleReader := bufio.NewReaderSize(os.Stdin, 1)
			_, _ = consoleReader.ReadByte()
			os.Exit(0)
		}
	}()

	flag.Parse()
	log.Println("[Main]", "读取配置文件中")
	importConfig()
	log.Println("[Main]", "启动中，AcLiveChat，", Version)
	log.Println("[Main]", "头像缓存时间：", AvatarRefreshRate, "秒")
	startHttpServer()
}

func importConfig() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("[Main]", "发生配置文件错误：", r)
			log.Println("[Main]", "跳过配置文件使用默认值")
		}
	}()

	var config = parseConfig.New("config.json")
	var BanWords = config.Get("BanWords").([]interface{})
	var UserMark = config.Get("UserMarks").(map[string]interface{})
	for _, v := range BanWords {
		BanString = append(BanString, v.(string))
	}
	for k, v := range UserMark {
		UserMarks[k] = v.(string)
	}

	LoveText = config.Get("LoveText").(string)
	FollowText = config.Get("FollowText").(string)
	JoinText = config.Get("JoinText").(string)
	QuitText = config.Get("QuitText").(string)
	AvatarRefreshRate = int(config.Get("AvatarRefreshRate").(float64))
}
