package danmu

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

type CmdType int

const (
	ADD CmdType = 1 + iota
	DEL
)

type Command struct {
	cmd CmdType
	url string
}
