package danmu

import (
    "strings"
)


var danmu *Danmu

type Danmu struct {
    roomMap map[string]string
}

func init() {
    danmu = &Danmu{roomMap: make(map[string]string)}
}

func New() *Danmu {
    return danmu
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
        p.Run()
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
