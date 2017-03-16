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
	"sync"
	"time"
)

const configUrl string = "http://www.quanmin.tv/site/route?time=%d"
const zzzzUrl string = "http://www.quanmin.tv/json/rooms/%s/info.json?t=%d"

func NewQuanmin(callback FuncType) *QuanminClient {
	return &QuanminClient{
		Rooms:    make(map[string]*QuanminRoom),
		Callback: callback}
}

type QuanminClient struct {
	Rooms    map[string]*QuanminRoom
	Lock     sync.RWMutex
	Callback FuncType
}

type QuanminRoom struct {
	url   string
	ip    string
	port  int
	uid   int
	conn  net.Conn
	alive bool
}

func (q *QuanminClient) Has(url string) bool {
	key := GenRoomKey(TrimUrl(url))
	q.Lock.RLock()
	defer q.Lock.RUnlock()
	if _, ok := q.Rooms[key]; ok {
		return true
	}
	return false
}

func (q *QuanminClient) Add(url string) {
	key := GenRoomKey(TrimUrl(url))
	q.Lock.Lock()
	defer q.Lock.Unlock()
	if _, ok := q.Rooms[key]; !ok {
		room := new(QuanminRoom)
		room.url = url
		room.uid, _ = strconv.Atoi(GetRoomId(url))
		room.alive = false
		q.Rooms[key] = room
	}
}

func (q *QuanminClient) Del(url string) {
	key := GenRoomKey(TrimUrl(url))
	q.Lock.Lock()
	defer q.Lock.Unlock()
	if _, ok := q.Rooms[key]; ok {
		q.Lock.RLock()
		defer q.Lock.RLock()
		delete(q.Rooms, key)
	}
}

func (q *QuanminClient) Online(url string) bool {
	return true
}

func (q *QuanminClient) Run(stop chan int) {
	//go c.Heartbeat(2 * 60)

	for {
		q.Lock.RLock()
		for _, room := range q.Rooms {
			if !room.alive {
				go q.Worker(room)
			}
		}
		q.Lock.RUnlock()

		// sleep to wait for new room
		time.Sleep(time.Second * 60)
	}

	stop <- 1
}

func (q *QuanminClient) Worker(p interface{}) {
	err := q.Prepare(p)
	if err != nil {
		log.Println("Prepare error", err)
		return
	}

	err = q.Connect(p)
	if err != nil {
		log.Println("Connect error", err)
		return
	}

	q.PullMsg(p, q.Callback)
}

func (q *QuanminClient) Prepare(p interface{}) error {
	room := p.(*QuanminRoom)

	url := fmt.Sprintf(configUrl, int(time.Now().Unix()))
	buf, _ := HttpGet(url, nil)

	a := int32(binary.BigEndian.Uint32(buf[0:4])) ^ 172
	b := int32(binary.BigEndian.Uint32(buf[4:8])) ^ 172
	c := int32(binary.BigEndian.Uint32(buf[8:12])) ^ 172
	d := int32(binary.BigEndian.Uint32(buf[12:16])) ^ 172

	room.ip = fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
	room.port = 9098
	roomId := GetRoomId(TrimUrl(room.url))

	z, err := strconv.Atoi(roomId)
	if err == nil {
		room.uid = z
	} else {
		buf, _ = HttpGet(fmt.Sprintf(zzzzUrl, roomId, int(time.Now().Unix()/50)), nil)
		js, _ := simplejson.NewJson(buf)
		room.uid = js.Get("uid").MustInt()
	}

	return nil
}

func (q *QuanminClient) Connect(p interface{}) error {
	room := p.(*QuanminRoom)

	addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", room.ip, room.port))
	if err != nil {
		return err
	}
	room.conn, err = net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	room.alive = true

	msg := q.genWriteBuffer(p)
	room.conn.Write(msg.Bytes())

	return nil

}

func (q *QuanminClient) Heartbeat(seconds int) error {
	t := time.NewTicker(time.Duration(seconds*1000) * time.Millisecond)
	for {
		<-t.C
		for _, room := range q.Rooms {
			if room.alive {
				log.Println("Panda Heartbeat", room.uid)
				var msg bytes.Buffer
				msg.Write([]byte{0x00, 0x06, 0x00, 0x00})
				err := q.PushMsg(room, msg.Bytes())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (q *QuanminClient) PushMsg(p interface{}, msg []byte) error {
	room := p.(*QuanminRoom)
	room.conn.Write(msg)
	return nil
}

func (q *QuanminClient) PullMsg(p interface{}, f FuncType) error {
	room := p.(*QuanminRoom)
	recvBuffer := make([]byte, 2048)
	for {
		n, err := room.conn.Read(recvBuffer)
		if n == 0 || err != nil {
			continue
		}

		msgs := q.parse(p, recvBuffer)
		for _, v := range msgs {
			f(v)
		}
	}
	return nil
}

func (q *QuanminClient) genWriteBuffer(p interface{}) bytes.Buffer {
	room := p.(*QuanminRoom)

	var buffer bytes.Buffer
	buffer.WriteString("{\n")
	buffer.WriteString("   \"os\" : 135,\n")
	buffer.WriteString("   \"pid\" : 10003,\n")
	buffer.WriteString(fmt.Sprintf("   \"rid\" : \"%d\",\n", room.uid))
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

func (q *QuanminClient) parse(p interface{}, data []byte) []*Msg {
	room := p.(*QuanminRoom)

	var result []*Msg

	reg := regexp.MustCompile(`({"ver".*?"cid":1})`)
	contents := reg.FindAllString(string(data), -1)
	for _, item := range contents {
		js, err := simplejson.NewJson([]byte(item))
		if err != nil {
			continue
		}
		json, err := js.Get("chat").Get("json").String()
		if err != nil {
		}

		js, _ = simplejson.NewJson([]byte(json))
		name := js.Get("user").Get("nick").MustString()
		if err != nil {
			continue
		}

		text, err := js.Get("text").String()
		if err != nil {
			continue
		}
		roomId := strconv.Itoa(room.uid)
		result = append(result, NewMsg("quanmin", roomId, name, text))
	}

	return result
}
