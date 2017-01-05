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
	"time"
)

func NewDouyu(callback FuncType) *DouyuClient {
	return &DouyuClient{
		rooms:    make(map[string]*dparams),
		callback: callback,
		stop:     make(chan int)}
}

type DouyuClient struct {
	rooms    map[string]*dparams
	callback FuncType
	stop     chan int
}

type dparams struct {
	url  string
	room string
	conn net.Conn
}

func (d *DouyuClient) Has(url string) bool {
	key := GenRoomKey(url)
	if _, ok := d.rooms[key]; ok {
		return true
	}
	return false
}

func (d *DouyuClient) Add(url string) {
	if d.Has(url) {
		return
	}
	key := GenRoomKey(TrimUrl(url))
	p := new(dparams)
	p.url = url
	p.room = d.getRoomId(url)
	d.rooms[key] = p
}

func (d *DouyuClient) Heartbeat(p interface{}) error {
	return nil
}

func (d *DouyuClient) Online(url string) bool {
	// TODO
	return true
}

func (d *DouyuClient) Remove(url string) {
	key := GenRoomKey(TrimUrl(url))

	if _, ok := d.rooms[key]; ok {
		delete(d.rooms, key)
	}
}

func (d *DouyuClient) Run(stop chan int) {
	for _, param := range d.rooms {
		// TODO
		go d.worker(param)
	}

	for i := 0; i < len(d.rooms); i++ {
		<-d.stop
	}
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

	d.PullMsg(p, d.callback)

	d.stop <- 1
}

func (d *DouyuClient) Prepare(p interface{}) error {
	return nil
}

func (d *DouyuClient) Connect(p interface{}) error {
	mparam := p.(*dparams)

	addr, _ := net.ResolveTCPAddr("tcp4", "openbarrage.douyutv.com:8601")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	mparam.conn = conn

	return nil
}

func (d *DouyuClient) PushMsg(p interface{}, msg []byte) error {
	mparam := p.(*dparams)

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

	_, err := mparam.conn.Write(content.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (d *DouyuClient) PullMsg(p interface{}, f FuncType) error {
	mparam := p.(*dparams)

	tmpl := "type@=loginreq/roomid@=%s/"
	msg := fmt.Sprintf(tmpl, mparam.room)
	err := d.PushMsg(mparam, []byte(msg))
	if err != nil {
		return err
	}

	recvBuffer := make([]byte, 2048)

	mparam.conn.Read(recvBuffer)

	log.Println(string(recvBuffer))

	tmpl = "type@=joingroup/rid@=%s/gid@=-9999/"
	msg = fmt.Sprintf(tmpl, mparam.room)
	err = d.PushMsg(mparam, []byte(msg))
	if err != nil {
		return err
	}

	tmpl = "type@=keeplive/tick@=" + strconv.FormatInt(time.Now().Unix(), 10)
	err = d.PushMsg(mparam, []byte(tmpl))
	if err != nil {
		return err
	}

	//recvBuffer := make([]byte, 2048)

	for {
		mparam.conn.Read(recvBuffer)
		msg := d.parse(mparam, recvBuffer)
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
	mparam := p.(*dparams)

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
		name, _ := sj.Get("nn").String()
		txt, _ := sj.Get("txt").String()
		return NewMsg("douyu", mparam.room, name, txt)
	}

	return NewOther("douyu", mparam.room, string(content))
}
