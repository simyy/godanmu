package danmu

type IDanmuClient interface {
	//
	Add()
	Has() bool
	Online() bool
	//
	Run()
	Prepare()
	Connect()
	PushMsg()
	ReadMsg()
}

type Danmu struct {
	channel chan int
	clients map[int]interface{}
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
}
