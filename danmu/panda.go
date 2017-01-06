package danmu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/bitly/go-simplejson"
	"log"
	"net"
	"strconv"
	"time"
)

func NewPanda(callback FuncType) *PandaClient {
	return &PandaClient{
		rooms:    make(map[string]*params),
		callback: callback,
		stop:     make(chan int)}
}

type PandaClient struct {
	rooms    map[string]*params
	callback FuncType
	stop     chan int
}

type params struct {
	url      string
	room     string
	conn     net.Conn
	u        string
	k        int
	t        int
	ts       int
	sign     string
	authtype string
	addrlist []string
}

func (c *PandaClient) Has(key string) bool {
	if _, ok := c.rooms[key]; ok {
		return true
	}
	return false
}

func (c *PandaClient) Add(url string) {
	key := GenRoomKey(TrimUrl(url))
	if _, ok := c.rooms[key]; !ok {
		p := new(params)
		p.url = url
		p.room = GetRoomId(url)
		c.rooms[key] = p
	}
}

func (c *PandaClient) Online(url string) bool {
	val := make(map[string]string)
	val["roomid"] = GetRoomId(url)
	val["_"] = strconv.FormatInt(time.Now().Unix(), 10)
	val["pub_key"] = ""
	statusUrl := "http://www.panda.tv/api_room"
	body, err := HttpGet(statusUrl, val)
	if err != nil {
		return false
	}

	js, _ := simplejson.NewJson(body)
	status := js.Get("data").Get("videoinfo").Get("status").MustString()
	return status == "2"
}

func (c *PandaClient) Remove(url string) {
	key := GenRoomKey(TrimUrl(url))
	if _, ok := c.rooms[key]; ok {
		delete(c.rooms, key)
	}
}

func (c *PandaClient) Run(stop chan int) {
	for _, p := range c.rooms {
		go c.worker(p)
	}

	go c.Task()

	for i := 0; i < len(c.rooms); i++ {
		<-c.stop
	}

	stop <- 1
}

func (c *PandaClient) Task() {
	t := time.NewTicker(time.Second * 2)
	for {
		<-t.C
		log.Println("task")
		for _, param := range c.rooms {
			c.Heartbeat(param)
		}
	}

}

func (c *PandaClient) worker(p interface{}) {
	err := c.Prepare(p)
	if err != nil {
		log.Println("Prepare error", err)
		return
	}

	err = c.Connect(p)
	if err != nil {
		log.Println("Connect error", err)
		return
	}

	c.PullMsg(p, c.callback)
	c.stop <- 1
}

func (c *PandaClient) Prepare(p interface{}) error {
	mparam := p.(*params)

	val := make(map[string]string)
	val["roomid"] = GetRoomId(mparam.url)
	val["_"] = strconv.FormatInt(time.Now().Unix(), 10)
	roomUrl := "http://www.panda.tv/ajax_chatroom"
	body, err := HttpGet(roomUrl, val)
	if err != nil {
		return err
	}

	js, err := simplejson.NewJson(body)
	if err != nil {
		return err
	}

	val["_"] = strconv.FormatInt(time.Now().Unix(), 10)
	val["rid"] = strconv.Itoa(js.Get("data").Get("rid").MustInt())
	val["retry"] = strconv.Itoa(0)
	val["sign"] = js.Get("data").Get("sign").MustString()
	val["ts"] = strconv.Itoa(js.Get("data").Get("ts").MustInt())
	infoUrl := "http://api.homer.panda.tv/chatroom/getinfo"
	body, err = HttpGet(infoUrl, val)
	if err != nil {
		return err
	}

	js, err = simplejson.NewJson(body)
	if err != nil {
		return err
	}

	mparam.u = fmt.Sprintf("%d@%s", js.Get("data").Get("rid").MustInt(),
		js.Get("data").Get("appid").MustString())
	mparam.k = 1
	mparam.t = 300
	mparam.ts = js.Get("data").Get("ts").MustInt()
	mparam.sign = js.Get("data").Get("sign").MustString()
	mparam.authtype = js.Get("data").Get("authType").MustString()
	mparam.addrlist = js.Get("data").Get("chat_addr_list").MustStringArray()

	return nil
}

func (c *PandaClient) Connect(p interface{}) error {
	mparam := p.(*params)
	addr, err := net.ResolveTCPAddr("tcp4", mparam.addrlist[0])
	if err != nil {
		return err
	}
	mparam.conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	msg := genWriteBuffer(mparam)
	mparam.conn.Write(msg.Bytes())
	// 写入呼吸包
	mparam.conn.Write([]byte{0x00, 0x06, 0x00, 0x00})

	return nil
}

func (c *PandaClient) Heartbeat(p interface{}) error {
	var msg bytes.Buffer
	msg.Write([]byte{0x00, 0x06, 0x00, 0x00})
	err := c.PushMsg(p, msg.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (c *PandaClient) PushMsg(p interface{}, msg []byte) error {
	mparam := p.(*params)
	mparam.conn.Write(msg)
	return nil
}

func (c *PandaClient) PullMsg(p interface{}, f FuncType) error {
	mparam := p.(*params)
	recvBuffer := make([]byte, 2048)
	for {
		n, err := mparam.conn.Read(recvBuffer)
		if n == 0 || err != nil {
			continue
		}

		prefix := []byte{0x00, 0x06, 0x00, 0x03}
		if bytes.HasPrefix(recvBuffer, prefix) {
			bufferSize := binary.BigEndian.Uint32(recvBuffer[11:15])
			msg := parse(p, recvBuffer[15+16:15+bufferSize])
			f(msg)
		}
	}
	return nil
}

func genWriteBuffer(p *params) bytes.Buffer {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("u:%s", p.u))
	buffer.WriteString("\n")
	buffer.WriteString(fmt.Sprintf("k:%d", p.k))
	buffer.WriteString("\n")
	buffer.WriteString(fmt.Sprintf("t:%d", p.t))
	buffer.WriteString("\n")
	buffer.WriteString(fmt.Sprintf("ts:%d", p.ts))
	buffer.WriteString("\n")
	buffer.WriteString(fmt.Sprintf("sign:%s", p.sign))
	buffer.WriteString("\n")
	buffer.WriteString(fmt.Sprintf("authtype:%s", p.authtype))

	length := len(buffer.Bytes())
	int16buf := new(bytes.Buffer)
	binary.Write(int16buf, binary.BigEndian, uint16(length))

	var msg bytes.Buffer
	// 消息头
	msg.Write([]byte{0x00, 0x06, 0x00, 0x02})
	// 写入数据长度
	msg.Write(int16buf.Bytes())
	// 写入数据内容
	msg.Write(buffer.Bytes())
	// 呼吸包
	msg.Write([]byte{0x00, 0x06, 0x00, 0x00})

	return msg
}

func parse(p interface{}, data []byte) *Msg {
	mparam := p.(*params)
	js, _ := simplejson.NewJson(data)
	_type, _ := js.Get("type").String()
	if _type == "1" {
		name := js.Get("data").Get("from").Get("nickName").MustString()
		text := js.Get("data").Get("content").MustString()
		return NewMsg("panda", mparam.room, name, text)
	}
	return NewOther("panda", mparam.room, string(data))
}
