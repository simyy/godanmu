package danmu

import (
    "log"
    "time"
    "fmt"
    "net"
    "bytes"
    "encoding/binary"
    "strconv"
    "github.com/bitly/go-simplejson"
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

func (d *Danmu) Register(roomUrl string) {
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

func (d *Danmu) Run() {
    if len(d.roomMap) == 0 {
        return
    }

    for _, v := range d.roomMap {
        p := &PandaClient{roomUrl: v}
        p.LoadConfig()
        p.InitSocket()
    }
}

type PandaClient struct {
    roomUrl        string
    u              string
    k              int
    t              int
    ts             int
    sign           string
    authtype       string
    chat_addr_list []string
}

type ChatroomResp struct {
    errno  int
    errmsg string
    data   Chatroom
}

type GetinfoResp struct {
    errno  int
    errmsg string
    data   Getinfo
}

type Chatroom struct {
    sign   string
    roomid int
    rid    int
    ts     int
}

type Getinfo struct {
    appid          string
    rid            int
    sign           string
    authType       string
    ts             int
    chat_addr_list []string
}

func (p *PandaClient) LoadConfig() bool {
    log.Println("加载网络配置")

    roomid := GetRoomId(p.roomUrl)
    url := "http://www.panda.tv/ajax_chatroom"
    params := make(map[string]string)
    params["roomid"] = roomid
    params["_"] = strconv.FormatInt(time.Now().Unix(), 10)
    body, err := HttpGet(url, params)
    if err != nil {
        log.Println("HttpGet error=", err)
        return false
    }

    js, err := simplejson.NewJson(body)
    if err != nil {
        log.Println("json error=", err)
        return false
    }

    errno, _ := js.Get("errno").Int()
    if errno != 0 {
        log.Println(js.Get("errmsg"))
        return false
    }

    time.Sleep(2 * time.Second)

    url = "http://api.homer.panda.tv/chatroom/getinfo"
    rid, _ := js.Get("data").Get("rid").Int()
    ts, _   := js.Get("data").Get("ts").Int()
    params1 := make(map[string]string)
    params1["rid"]     = strconv.Itoa(rid)
    params1["roomid"]  = roomid
    params1["retry"]   = strconv.Itoa(0)
    params1["sign"], _ = js.Get("data").Get("sign").String()
    params1["ts"]  = strconv.Itoa(ts)
    params1["_"]       = strconv.FormatInt(time.Now().Unix(), 10)
    body, err = HttpGet(url, params1)
    if err != nil {
        log.Println("HttpGet error=", err)
        return false
    }

    js, err = simplejson.NewJson(body)
    if err != nil {
        log.Println("json error=", err)
        return false
    }

    errno, _ = js.Get("errno").Int()
    if errno != 0 {
        log.Println(js.Get("errmsg"))
        return false
    }

    rid, _   = js.Get("data").Get("rid").Int()
    appid, _ := js.Get("data").Get("appid").String()
    log.Println(appid)
    p.u = fmt.Sprintf("%d@%s", rid, appid)
    p.k = 1
    p.t = 300
    p.ts, _ = js.Get("data").Get("ts").Int()
    p.sign, _ = js.Get("data").Get("sign").String()
    p.authtype, _ = js.Get("data").Get("authType").String()
    p.chat_addr_list, _ = js.Get("data").Get("chat_addr_list").StringArray()
    return true
}

//整形转换成字节4位
func IntToBytes4(n int) []byte {
    m := int32(n)
    bytesBuffer := bytes.NewBuffer([]byte{})
    binary.Write(bytesBuffer, binary.BigEndian, m)

    gbyte := bytesBuffer.Bytes()
    //c++ 高低位转换
    k := 4
    x := len(gbyte)
    nb := make([]byte, k)
    for i := 0; i < k; i++ {
        nb[i] = gbyte[x-i-1]
    }
    return nb
}

func (p *PandaClient) InitSocket() {
    log.Println("初始化网络连接")

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

    log.Println([]byte(buffer.String()))

    length := len(buffer.Bytes())
    int16buf := new(bytes.Buffer)
    i := uint16(length)
    binary.Write(int16buf,binary.BigEndian,i)

    var msg bytes.Buffer
    msg.Write([]byte{0x00, 0x06, 0x00, 0x02})
    msg.Write(int16buf.Bytes())
    msg.Write(buffer.Bytes()) 
    msg.Write([]byte{0x00, 0x06, 0x00, 0x00})

    addr, _ := net.ResolveTCPAddr("tcp4", p.chat_addr_list[0])
    log.Println("服务器地址:", addr)
    conn, _ := net.DialTCP("tcp", nil, addr)
    conn.Write(msg.Bytes())
    conn.Write([]byte{0x00, 0x06, 0x00, 0x00})

    recvBuffer := make([]byte, 2048)  
    for {
        n, err := conn.Read(recvBuffer)
        if n == 0 || err != nil {
            continue
        }

        //log.Println("收到字节数:", n)
        //log.Println("收到字节为:", recvBuffer)

        prefix := []byte{0x00, 0x06, 0x00, 0x03}

        if bytes.HasPrefix(recvBuffer, prefix) {
            bufferSize := binary.BigEndian.Uint32(recvBuffer[11:15])
            //log.Println("收到内容长度:", bufferSize)
            //log.Println(recvBuffer[15:])
            //log.Println("收到弹幕内容:", recvBuffer[15+16:15+bufferSize])
            //log.Println("收到弹幕内容:", string(recvBuffer[15+16:15+bufferSize]))
            p.Parse(recvBuffer[15+16:15+bufferSize])
        }
    }
}

func (p *PandaClient) Parse(data []byte) {
    js, _ := simplejson.NewJson(data)
    _type, _ := js.Get("type").String()
    name, _ := js.Get("data").Get("from").Get("nickName").String()
    content, _ := js.Get("data").Get("content").String()
    if _type == "1" {
        log.Println(name, content)
    } else {
        log.Println(string(data))
    }
}
