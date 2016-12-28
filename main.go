package main

import (
	"github.com/yxd123/simDanmu/danmu"
	"log"
)

func main() {
	log.Println("程序启动")

	c := make(chan int)

	danmu := danmu.New(c)
	// danmu.Add("http://www.panda.tv/573130")
	// danmu.Add("http://www.douyu.com/976537")

	danmu.Add("http://www.quanmin.tv/v/15")
	danmu.Run()

	for i := 0; i < 2; i++ {
		<-c
	}

	log.Println("程序结束")
}
