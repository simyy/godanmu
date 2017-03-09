package danmu

import (
	"log"
	"strings"
)

type FuncType func(*Msg)

type IDanmuClient interface {
	Add(url string)
	Del(url string)
	Run(c chan int)
	Online(url string) bool
	Prepare(p interface{}) error
	Connect(p interface{}) error
	Heartbeat(seconds int) error
	PushMsg(p interface{}, msg []byte) error
	PullMsg(p interface{}, f FuncType) error
}

type Danmu struct {
	stop    chan int
	clients map[string]IDanmuClient
}

func New(f FuncType) *Danmu {
	clients := make(map[string]IDanmuClient)
	clients["panda"] = NewPanda(f)
	clients["douyu"] = NewDouyu(f)
	//clients["huomao"] = NewHuomao()
	//clients["quanmin"] = NewQuanmin()

	danmu := &Danmu{
		stop:    make(chan int),
		clients: clients}

	return danmu
}

func (d *Danmu) Add(url string) {
	client := d.match(url)
	client.Add(url)
}

func (d *Danmu) Del(url string) {
	client := d.match(url)
	client.Del(url)
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

func (d *Danmu) match(url string) IDanmuClient {
	for k, v := range d.clients {
		if strings.Contains(url, k) {
			return v
		}
	}

	return nil
}
