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
	"strings"
	"sync"
	"time"
)

func NewDouyu(callback FuncType) *DouyuClient {
	return &DouyuClient{
		Rooms:    make(map[string]*DouyuRoom),
		Callback: callback}
}

type DouyuClient struct {
	Rooms    map[string]*DouyuRoom
	Lock     sync.RWMutex
	Callback FuncType
}

type DouyuRoom struct {
	url   string
	room  string
	alive bool
	conn  net.Conn
}

func (d *DouyuClient) Has(url string) bool {
	key := GenRoomKey(url)
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	if _, ok := d.Rooms[key]; ok {
		return true
	}
	return false
}

func (d *DouyuClient) Add(url string) {
	if d.Has(url) {
		return
	}
	key := GenRoomKey(TrimUrl(url))
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	p := new(DouyuRoom)
	p.url = url
	p.room = d.getRoomId(url)
	p.alive = false
	d.Rooms[key] = p
}

func (d *DouyuClient) Del(url string) {
	key := GenRoomKey(TrimUrl(url))
	d.Lock.RLock()
	defer d.Lock.RUnlock()
	if _, ok := d.Rooms[key]; ok {
		delete(d.Rooms, key)
	}
}

func (d *DouyuClient) Online(url string) bool {
	tmpl := "http://open.douyucdn.cn/api/RoomApi/room/%s"
	configUrl := fmt.Sprintf(tmpl, GetRoomId(url))
	body, err := HttpGet(configUrl, nil)

	js, _ := simplejson.NewJson(body)
	if js.Get("error").MustInt() != 0 {
		return false
	}

	if js.Get("data").Get("room_status").MustString() != "1" {
		return false
	}

	return true
}

func (d *DouyuClient) Run(stop chan int) {
	go d.Heartbeat(30)

	for {
		d.Lock.RLock()
		for _, room := range d.Rooms {
			if !room.alive {
				go d.worker(room)
			}
		}
		d.Lock.RUnlock()

		time.Sleep(time.Second * 60)
	}

	stop <- 1

}

func (d *DouyuClient) worker(p interface{}) {
	err := d.Prepare(p)
	if err != nil {
		log.Println("Prepare error", err)
		return
	}

	err = d.Connect(p)
	if err != nil {
		log.Println("Connect error", err)
		return
	}

	d.PullMsg(p, d.Callback)
}

func (d *DouyuClient) Prepare(p interface{}) error {
	return nil
}

func (d *DouyuClient) Connect(p interface{}) error {
	room := p.(*DouyuRoom)

	addr, _ := net.ResolveTCPAddr("tcp4", "openbarrage.douyutv.com:8601")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	room.alive = true
	room.conn = conn

	return nil
}

func (d *DouyuClient) Heartbeat(seconds int) error {
	t := time.NewTicker(time.Duration(seconds*1000) * time.Millisecond)
	for {
		<-t.C
		for _, room := range d.Rooms {
			if room.alive {
				log.Println("Douyu Hearbeat", room.room)
				tmpl := "type@=keeplive/tick@=%s/"
				msg := fmt.Sprintf(tmpl, room.room)
				err := d.PushMsg(room, []byte(msg))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (d *DouyuClient) PushMsg(p interface{}, msg []byte) error {
	room := p.(*DouyuRoom)

	s := 9 + len(msg)
	int16buf := new(bytes.Buffer)
	binary.Write(int16buf, binary.LittleEndian, uint32(s))
	binary.Write(int16buf, binary.LittleEndian, uint32(s))

	header := []byte{0xb1, 0x02, 0x00, 0x00}

	var content bytes.Buffer
	content.Write(int16buf.Bytes())
	content.Write(header)
	content.Write(msg)
	content.Write([]byte{0x00})

	_, err := room.conn.Write(content.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (d *DouyuClient) PullMsg(p interface{}, f FuncType) error {
	room := p.(*DouyuRoom)

	tmpl := "type@=loginreq/roomid@=%s/"
	msg := fmt.Sprintf(tmpl, room.room)
	err := d.PushMsg(room, []byte(msg))
	if err != nil {
		return err
	}

	recvBuffer := make([]byte, 2048)

	room.conn.Read(recvBuffer)

	tmpl = "type@=joingroup/rid@=%s/gid@=-9999/"
	msg = fmt.Sprintf(tmpl, room.room)
	err = d.PushMsg(room, []byte(msg))
	if err != nil {
		return err
	}

	tmpl = "type@=keeplive/tick@=" + strconv.FormatInt(time.Now().Unix(), 10)
	err = d.PushMsg(room, []byte(tmpl))
	if err != nil {
		return err
	}

	for {
		room.conn.Read(recvBuffer)
		msg := d.parse(room, recvBuffer)
		f(msg)
	}

	return nil
}

func (d *DouyuClient) getRoomId(url string) string {
	tmpl := "http://open.douyucdn.cn/api/RoomApi/room/%s"
	configUrl := fmt.Sprintf(tmpl, GetRoomId(url))
	body, err := HttpGet(configUrl, nil)
	if err != nil {
		return ""
	}

	js, _ := simplejson.NewJson(body)
	if js.Get("error").MustInt() != 0 {
		return ""
	}

	if js.Get("data").Get("room_status").MustString() != "1" {
		return ""
	}

	return js.Get("data").Get("room_id").MustString()
}

func (d *DouyuClient) parse(p interface{}, data []byte) *Msg {
	room := p.(*DouyuRoom)

	content := string(data)
	content = strings.Replace(content, "@=", "\":\"", -1)
	content = strings.Replace(content, "/", "\",\"", -1)
	content = strings.Replace(content, "@A", "@", -1)
	content = strings.Replace(content, "@S", "/", -1)

	reg := regexp.MustCompile(`type":"chatmsg",(.*?),"el":""`)
	contents := reg.FindAllString(content, -1)
	for _, item := range contents {
		tmp := "{\"" + item + "}"
		sj, _ := simplejson.NewJson([]byte(tmp))
		name := sj.Get("nn").MustString()
		txt := sj.Get("txt").MustString()
		return NewMsg("douyu", room.room, name, txt)
	}

	return NewOther("douyu", room.room, string(content))
}
