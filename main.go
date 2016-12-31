package main

import (
	"github.com/yxd123/simDanmu/danmu"
	"log"
)

func Handler(msg *danmu.Msg) {
	if !msg.IsMsg() {
		log.Println(msg.Site, msg.Room, msg.Other)
	} else {
		log.Println(msg.Site, msg.Room, msg.Name, msg.Text)
	}
}

func main() {
	danmu := danmu.New(Handler)
	danmu.Add("http://www.panda.tv/638744")
	danmu.Add("http://www.panda.tv/434939")
	danmu.Run()
}
