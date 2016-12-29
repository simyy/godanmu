package danmu

import (
	"log"
	"regexp"
)

type IDanmuClient interface {
	Add()
	Has() bool
	Online() bool
	Run()
	Prepare()
	Connect()
	PushMsg(msg []byte)
	PullMsg(callback func(msg []byte))
}

type Danmu struct {
	channel chan int
	clients map[int]interface{}
}

func (d *Danmu) match(roomUrl string) IDanmuClient {
	reg := regexp.MustCompile(`.*\W+(\w*)\.[tvcom]{2,3}.*`)
	key := reg.FindString(roomUrl)
	if _, ok := d.clients[key]; ok {
		return d.clients[key]
	}

	return nil
}

func New(channel chan int) *Danmu {
	clients := make(map[string]interface{})
	clients["panda"] = NewPanda()
	clients["douyu"] = NewDouyu()
	clients["huomao"] = NewHuomao()
	clients["quanmin"] = NewQuanmin()

	danmu := &Danmu{
		channel: channel,
		clients: clients}

	return danmu
}

func (d *Danmu) Push(roomUrl string) {
	roomUrl = TrimUrl(roomUrl)
	key := GenRoomKey(roomUrl)
	for _, client := range d.clients {
		if _, ok := client[key]; ok {
			log.Println("url exist:", roomUrl)
			return
		}
	}

	client := d.match(roomUrl)
	client.Add(roomUrl)
}
