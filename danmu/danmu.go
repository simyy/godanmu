package danmu

import (
	"strings"
)

type Danmu struct {
	channel chan int
	roomMap map[string]string
}

func New(channel chan int) *Danmu {
	return &Danmu{channel: channel, roomMap: make(map[string]string)}
}

func (d *Danmu) Add(roomUrl string) {
	roomUrl = TrimUrl(roomUrl)
	key := GenRoomKey(roomUrl)
	if _, ok := d.roomMap[key]; !ok {
		d.roomMap[key] = roomUrl
	}
}

func (d *Danmu) Delete(roomUrl string) {
	roomUrl = TrimUrl(roomUrl)
	key := GenRoomKey(roomUrl)
	if _, ok := d.roomMap[key]; ok {
		delete(d.roomMap, key)
	}
}

func worker(roomUrl string) {
	if strings.Contains(roomUrl, "panda.tv") {
		p := &PandaClient{url: roomUrl}
		go p.Run()
	}

	if strings.Contains(roomUrl, "douyu.com") {
		p := &DouyuClient{url: roomUrl}
		go p.Run()
	}

	if strings.Contains(roomUrl, "quanmin.tv") {
		p := &QuanminClient{url: roomUrl}
		go p.Run()
	}
}

func (d *Danmu) Run() {
	if len(d.roomMap) == 0 {
		return
	}

	for _, v := range d.roomMap {
		worker(v)
	}
}
