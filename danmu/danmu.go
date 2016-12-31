package danmu

import (
	"log"
	"strings"
)

type FuncType func(*Msg)

type IDanmuClient interface {
	Add(url string)
	Has(url string) bool
	Remove(url string)
	Online(url string) bool
	Run(c chan int)
	Prepare(p interface{}) error
	Connect(p interface{}) error
	Heartbeat(p interface{}) error
	PushMsg(p interface{}, msg []byte) error
	PullMsg(p interface{}, f FuncType) error
}

type Msg struct {
	Site  string `json:site`
	Room  string `json:room`
	Name  string `json:name`
	Text  string `json:text`
	Other string `json:other`
}

func NewMsg(site, room, name, text string) *Msg {
	return &Msg{
		Site:  site,
		Room:  room,
		Name:  name,
		Text:  text,
		Other: ""}
}

func NewOther(site, room, other string) *Msg {
	return &Msg{
		Site:  site,
		Room:  room,
		Name:  "",
		Text:  "",
		Other: other}
}

func (m *Msg) IsMsg() bool {
	if m.Other != "" {
		return false
	}
	return true
}

type Danmu struct {
	stop    chan int
	clients map[string]IDanmuClient
}

func New(f FuncType) *Danmu {
	clients := make(map[string]IDanmuClient)
	clients["panda"] = NewPanda(f)
	//clients["douyu"] = NewDouyu()
	//clients["huomao"] = NewHuomao()
	//clients["quanmin"] = NewQuanmin()

	danmu := &Danmu{
		stop:    make(chan int),
		clients: clients}

	return danmu
}

func (d *Danmu) match(url string) IDanmuClient {
	for k, v := range d.clients {
		if strings.Contains(url, k) {
			return v
		}
	}

	return nil
}

func (d *Danmu) Add(url string) {
	key := GenRoomKey(TrimUrl(url))
	for _, client := range d.clients {
		if client.Has(key) {
			return
		}
	}
	client := d.match(url)
	client.Add(url)
}

func (d *Danmu) Remove(url string) {
	key := GenRoomKey(TrimUrl(url))
	for _, client := range d.clients {
		if client.Has(key) {
			client.Remove(url)
			return
		}
	}
}

func (d *Danmu) Run() {
	log.Println("danmu start wait ...")
	for _, client := range d.clients {
		go client.Run(d.stop)
	}

	for i := 0; i < len(d.clients); i++ {
		<-d.stop
	}
	log.Println("Danmu stop byte!!!")
}
