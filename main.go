package main

import (
    "log"
    "github.com/yxd123/simDanmu/danmu"
)

func main() {
    log.Println("程序启动")
    danmu := danmu.New() 
    danmu.Register("http://www.panda.tv/777777")
    danmu.Run()
    log.Println("程序结束")
}
