package danmu

import (
    "log"
    "time"
    "fmt"
    "net"
    "bytes"
    "encoding/binary"
    "encoding/json"
    "strconv"
    "github.com/bitly/go-simplejson"
)

const roomUrl string = "http://www.panda.tv/ajax_chatroom"
const infoUrl string = "http://api.homer.panda.tv/chatroom/getinfo"


type PandaClient struct {
    url      string
    u        string
    k        int
    t        int
    ts       int
    sign     string
    authtype string
    addrlist []string
} 
    
type ChatroomData struct { 
    Sign   string `json:"sign"`
    Roomid int    `json:"roomid"`
    Rid    int    `json:"rid"`    
    Ts     int    `json:"ts"`
}

type Chatroom struct {
    Errno  int          `jsono:"errno"` 
    Data   ChatroomData `jsono:"data"` 
    Errmsg string       `jsono:"errmsg"` 
}

type GetInfoData struct {
    Rid      int      `json:"rid"`
    Appid    string   `json:"appid"` 
    AddrList []string `json:"chat_addr_list"` 
    Ts       int      `json:"ts"` 
    Sign     string   `json:"sign"` 
    AuthType string   `json:"authType"` 
}

type GetInfo struct {
    Errno  int          `jsono:"errno"` 
    Data   GetInfoData  `jsono:"data"` 
    Errmsg string       `jsono:"errmsg"` 
}

func (p *PandaClient) Run() {
    p.loadConfig()
    p.initSocket()
}

func getChatroomParams(roomId string) (Chatroom, error) {
    var cr Chatroom

    params := make(map[string]string)
    params["roomid"] = roomId
    params["_"] = strconv.FormatInt(time.Now().Unix(), 10)
    body, err := HttpGet(roomUrl, params)
    if err != nil {
        return cr, err
    }

    err = json.Unmarshal(body, &cr)
    if err != nil {
        return cr, err
    }

    return cr, nil
}

func getGetInfoParams(cr Chatroom) (GetInfo, error) {
    var gi GetInfo

    params := make(map[string]string)
    params["rid"]    = strconv.Itoa(cr.Data.Rid)
    params["roomid"] = strconv.Itoa(cr.Data.Roomid)
    params["retry"] = strconv.Itoa(0)
    params["sign"]  = cr.Data.Sign
    params["ts"]    = strconv.Itoa(cr.Data.Ts)
    params["_"]     = strconv.FormatInt(time.Now().Unix(), 10)

    body, err := HttpGet(infoUrl, params)
    if err != nil {
        return gi, err
    }

    log.Println(string(body))

    err = json.Unmarshal(body, &gi)
    if err != nil {
        return gi, err
    }

    log.Println(gi.Data)

    return gi, nil
}

func (p *PandaClient) loadConfig() bool {
    log.Println("加载网络配置 For PandaTV")

    roomId := GetRoomId(p.url)
    chatroom, err := getChatroomParams(roomId)
    if err != nil {
        return false
    }

    getInfo, err := getGetInfoParams(chatroom)
    if err != nil {
        return false
    }

    p.u = fmt.Sprintf("%d@%s", getInfo.Data.Rid, getInfo.Data.Appid)
    p.k = 1
    p.t = 300
    p.ts   = getInfo.Data.Ts
    p.sign = getInfo.Data.Sign
    p.authtype = getInfo.Data.AuthType
    p.addrlist = getInfo.Data.AddrList

    log.Println(p, getInfo.Data)

    return true
}

func (p *PandaClient) genWriteBuffer() bytes.Buffer {
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
    binary.Write(int16buf,binary.BigEndian, uint16(length))

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

func (p *PandaClient) initSocket() {
    log.Println("初始化网络连接 For PandaTV")

    msg := p.genWriteBuffer()

    addr, _ := net.ResolveTCPAddr("tcp4", p.addrlist[0])
    conn, _ := net.DialTCP("tcp", nil, addr)
    conn.Write(msg.Bytes())
    // 写入呼吸包
    conn.Write([]byte{0x00, 0x06, 0x00, 0x00})

    recvBuffer := make([]byte, 2048)  
    for {
        n, err := conn.Read(recvBuffer)
        if n == 0 || err != nil {
            continue
        }

        prefix := []byte{0x00, 0x06, 0x00, 0x03}
        if bytes.HasPrefix(recvBuffer, prefix) {
            bufferSize := binary.BigEndian.Uint32(recvBuffer[11:15])
            p.parse(recvBuffer[15+16:15+bufferSize])
        }
    }
}

func (p *PandaClient) parse(data []byte) {
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
