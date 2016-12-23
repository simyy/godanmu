package danmu

import (
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"time"
)

const configUrl string = "http://www.quanmin.tv/site/route?time=%d"
const zzzzUrl string = "http://www.quanmin.tv/json/rooms/%s/info.json?t=%d"

type QuanminClient struct {
	url  string
	ip   string
	port int
	uid  int
}

func (q *QuanminClient) Run() {
	q.loadConfig()
}

func (q *QuanminClient) loadConfig() {
	url := fmt.Sprintf(configUrl, int(time.Now().Unix()))
	buf, _ := HttpGet(url, nil)

	log.Println(buf)

	a := int32(binary.BigEndian.Uint32(buf[0:4])) ^ 172
	b := int32(binary.BigEndian.Uint32(buf[4:8])) ^ 172
	c := int32(binary.BigEndian.Uint32(buf[8:12])) ^ 172
	d := int32(binary.BigEndian.Uint32(buf[12:16])) ^ 172

	q.ip = fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
	q.port = 9098
	roomId := GetRoomId(TrimUrl(q.url))

	z, err := strconv.Atoi(roomId)
	if err == nil {
		q.uid = z
	} else {
		buf, _ = HttpGet(fmt.Sprintf(zzzzUrl, roomId, int(time.Now().Unix()/50)), nil)
		log.Println(string(buf))
	}
}
