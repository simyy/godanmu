package danmu

import (
    "log"
    "time"
    "fmt"
)

var danmu *Danmu

type Danmu struct {
    roomMap map[string]string
}

func init() {
    danmu := &Danmu{}
}

func New() *Danmu {
    return danmu
}

func (d *Danmu) Register(roomUrl string) {
    roomUrl = TrimUrl(roomUrl)
    key := GenRoomKey(roomUrl)
    if v, ok := d.roomMap.get(key); !ok {
        roomMap[key] = roomUrl
    }
}

func (d *Danmu) Delete(roomUrl string) {
    roomUrl = TrimUrl(roomUrl)
    key := GenRoomKey(roomUrl)
    if v, ok := d.roomMap.get(key); ok {
        delete(d.roomMap, key)
    }
}

type PandaClient struct {
    roomUrl  string
    u        string
    k        int
    t        int
    ts       int
    sing     string
    authtype string
}

type PandaResp struct {
    errno  int
    errmsg string
    data   string
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

func (p *PandaClient) Prepare() bool {
    body, err := HttpGet(p.roomUrl)
    if err != nil {
        log.Error(err)
        return false
    }

    roomid := GetRoomId(p.roomUrl)
    url := "http://www.panda.tv/ajax_chatroom"
    params := make(map[string]string)
    params["roomid"] = roomid
    params["_"] = time.Now().String()
    body, err := HttpGet(url, params)
    if err != nil {
        log.Error(err)
        return false
    }
    log.Debug(string(body))

    var resp PandaResp
    err := json.Unmarshal([]byte(body), &resp)
    if err != nil {
        log.Error(err)
        return
    }

    if resp.errno != 0 {
        log.Error(resp.errmsg)
        return
    }

    var chatroom Chatroom
    err := json.Unmarshal([]byte(resp.data), &chatroom)
    if err != nil {
        log.Error(err)
        return
    }

    url = "http://api.homer.panda.tv/chatroom/getinfo'"
    params1 = make([string]string)
    params1["rid"] = chatroom.rid
    params1["roomid"] = roomid
    params1["retry"] = 0
    params1["sign"] = chatroom.sign
    params1["ts"] = chatroom.ts
    params1["_"] = time.Now().String
    body, err = HttpGet(url, params1)
    if err != nil {
        log.Error(err)
        return false
    }
    log.Debug(string(body))


    var resp PandaResp
    err := json.Unmarshal([]byte(body), &resp)
    if err != nil {
        log.Error(err)
        return
    }

    if resp.errno != 0 {
        log.Error(resp.errmsg)
        return
    }

    var getinfo Getinfo
    err := json.Unmarshal([]byte(resp.data), &getinfo)
    if err != nil {
        log.Error(err)
        return
    }

    p.u = fmt.Sprintf("%s@%s", getinfo.rid, getinfo.appid)
    p.k = 1
    p.t = 300
    p.ts = getinfo.ts
    p.sign
    p.authtype = getinfo.authType
    p.chat_addr_list = getinfo.chat_addr_list

    log.info(p)
}
