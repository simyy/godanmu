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

type Douyu struct {
	rooms map[string]string
}

func NewDouyu() *Douyu {
	return &Douyu{rooms: make(map[string]string)}
}

func (d *Douyu) Has(url string) bool {
	key := GenRoomKey(url)
	if _, ok := d.rooms[key]; ok {
		return true
	}
	return false
}

func (d *Douyu) Add(url string) {
	if d.Has(url) {
		return
	}
	key := GenRoomKey(url)
	d.rooms[key] = url
}

func (d *Douyu) Online(url string) bool {
	// TODO
	return false
}

func (d *Douyu) Run() {
	for _, url := range d.rooms {
		roomId = d.getRoomId(url)
		if roomId < 0 {
			continue
		}

		// TODO
	}
}

func (d *Douyu) getRoomId() int {
	tmpl := "http://open.douyucdn.cn/api/RoomApi/room/%s"
	configUrl := fmt.Sprintf(tmpl, GetRoomId(d.url))
	body, err := HttpGet(configUrl, nil)
	if err != nil {
		return 0
	}

	js, _ := simplejson.NewJson(body)
	if _err, _ := js.Get("error").Int(); _err != 0 {
		return 0
	}

	if status, _ := js.Get("data").Get("room_status").String(); status != "1" {
		return 0
	}

	roomId, _ := js.Get("data").Get("room_id").String()
	result, _ := strconv.Atoi(roomId)
	return result
}

func push(conn net.Conn, msg []byte) {
	s := 9 + len(msg)

	int16buf := new(bytes.Buffer)
	binary.Write(int16buf, binary.LittleEndian, uint32(s))
	binary.Write(int16buf, binary.LittleEndian, uint32(s))

	//log.Println("int buffer", int16buf.Bytes())

	header := []byte{0xb1, 0x02, 0x00, 0x00}

	var content bytes.Buffer
	content.Write(int16buf.Bytes())
	content.Write(header)
	content.Write(msg)
	content.Write([]byte{0x00})

	//log.Println(content.Bytes())
	conn.Write(content.Bytes())
}

func pull(conn net.Conn) []byte {
	return nil
}

func (d *Douyu) initSocket() {
	log.Println("初始化网络连接 For DouyuTV")

	addr, _ := net.ResolveTCPAddr("tcp4", "openbarrage.douyutv.com:8601")
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println(d.roomId)
	tmpl := "type@=loginreq/roomid@=%d/"
	msg := fmt.Sprintf(tmpl, d.roomId)
	push(conn, []byte(msg))

	tmpl = "type@=joingroup/rid@=%d/gid@=-9999/"
	msg = fmt.Sprintf(tmpl, d.roomId)
	push(conn, []byte(msg))

	tmpl = "type@=keeplive/tick@=" + strconv.FormatInt(time.Now().Unix(), 10)
	push(conn, []byte(tmpl))

	recvBuffer := make([]byte, 2048)
	for {
		conn.Read(recvBuffer)
		// log.Println(n, recvBuffer)

		// bufferSize := binary.LittleEndian.Uint32(recvBuffer[0:4])
		// log.Println(bufferSize)
		d.parse(recvBuffer)
	}

}

func (d *Douyu) parse(data []byte) {
	content := string(data)
	content = strings.Replace(content, "@=", "\":\"", -1)
	content = strings.Replace(content, "/", "\",\"", -1)
	content = strings.Replace(content, "@A", "@", -1)
	content = strings.Replace(content, "@S", "/", -1)

	reg := regexp.MustCompile(`type":"chatmsg",(.*?),"el":""`)
	contents := reg.FindAllString(content, -1)
	for _, item := range contents {
		tmp := "{\"" + item + "}"
		//log.Println(tmp)
		sj, _ := simplejson.NewJson([]byte(tmp))
		name, _ := sj.Get("nn").String()
		txt, _ := sj.Get("txt").String()
		log.Println(name, txt)

	}
}
