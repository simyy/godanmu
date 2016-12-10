package danmu

import (
    "crypto/md5"
    "encoding/hex"
    "net/url"
    "net/http"
    "io/ioutil"
    "strings"
    "log"
)

func TrimUrl(url string) string {
    res := strings.TrimSpace(url) 
    return strings.TrimSuffix(res, "/")
}

func GetRoomId(url string) string {
    s := strings.Split(url, "/")
    return s[len(s) - 1]
}

// generate md5 for roomMap key
func GenRoomKey(roomUrl string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(roomUrl))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

// http GET
func HttpGet(urlStr string, params map[string]string) ([]byte, error) {
    u, _ := url.Parse(urlStr)
    if len(params) > 0 {
        q := u.Query()
        for k, v := range params {
            q.Set(k, v)
        }
        u.RawQuery = q.Encode()
    } 
    log.Println(u.String())
    res, err := http.Get(u.String())
    if err != nil {
        return nil, err
    }

    result, _ := ioutil.ReadAll(res.Body)
    res.Body.Close()

    return result, nil
}
