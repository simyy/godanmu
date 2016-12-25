package danmu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/bitly/go-simplejson"
	"log"
	"net"
	"regexp"
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
	q.initSocket()
}

func (q *QuanminClient) loadConfig() {
	url := fmt.Sprintf(configUrl, int(time.Now().Unix()))
	buf, _ := HttpGet(url, nil)

	log.Println("load config:", buf)

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
		js, _ := simplejson.NewJson(buf)
		q.uid, _ = js.Get("uid").Int()
	}
}

func (q *QuanminClient) genWriteBuffer() bytes.Buffer {
	var buffer bytes.Buffer
	buffer.WriteString("{\n")
	buffer.WriteString("   \"os\" : 135,\n")
	buffer.WriteString("   \"pid\" : 10003,\n")
	buffer.WriteString(fmt.Sprintf("   \"rid\" : \"%d\",\n", q.uid))
	buffer.WriteString("   \"timestamp\" : 78,\n")
	buffer.WriteString("   \"ver\" : 147\n}")

	length := len(buffer.Bytes())
	int32buf := new(bytes.Buffer)
	binary.Write(int32buf, binary.BigEndian, uint32(length))

	var msg bytes.Buffer
	msg.Write(int32buf.Bytes())
	msg.Write(buffer.Bytes())
	msg.Write([]byte{0x0a})

	return msg
}

func (q *QuanminClient) initSocket() {
	log.Println("初始化网络连接 for Quanmin")

	data := q.genWriteBuffer()

	addr, _ := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", q.ip, q.port))
	conn, _ := net.DialTCP("tcp", nil, addr)
	conn.Write(data.Bytes())

	recvBuffer := make([]byte, 2048)
	for {
		n, err := conn.Read(recvBuffer)
		if n == 0 || err != nil {
			continue
		}

		q.parse(recvBuffer)
	}
}

func (q *QuanminClient) parse(data []byte) {
	reg := regexp.MustCompile(`({"ver".*?"cid":1})`)
	contents := reg.FindAllString(string(data), -1)
	for _, item := range contents {
		js, err := simplejson.NewJson([]byte(item))
		if err != nil {
			log.Println("json error", err)
			continue
		}
		json, err := js.Get("chat").Get("json").String()
		if err != nil {
			log.Println("json err", err, string(item))
			continue
		}

		js, _ = simplejson.NewJson([]byte(json))
		nick, err := js.Get("user").Get("nick").String()
		if err != nil {
			log.Println("parse nick error", err, string(item))
			continue
		}

		text, err := js.Get("text").String()
		if err != nil {
			log.Println("parse nick error", err, string(item))
			continue
		}

		log.Println(nick, text)
	}
}
