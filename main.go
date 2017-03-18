package main

import (
	"github.com/yxd123/simDanmu/danmu"
	"log"
)

func Handler(msg *danmu.Msg) {
	if !msg.IsMsg() {
		//log.Println(msg.Site, msg.Room, msg.Other)
	} else {
		log.Println(msg.Site, msg.Room, msg.Name, msg.Text)
	}
}

func main() {
	danmu := danmu.New(Handler)
	//danmu.Add("http://www.panda.tv/777777")
	//danmu.Add("https://www.douyu.com/793400")
	//danmu.Add("http://www.quanmin.tv/3446603")
	danmu.Add("https://www.huomao.com/10519")
	danmu.Run()
}
