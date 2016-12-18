package main

import (
	"github.com/yxd123/simDanmu/danmu"
	"log"
)

func main() {
	log.Println("程序启动")
	danmu := danmu.New()
	// danmu.Add("http://www.panda.tv/471358")
	danmu.Add("http://www.douyu.com/yechui")
	danmu.Run()
	log.Println("程序结束")
}
