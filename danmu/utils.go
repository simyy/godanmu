package danmu

import (
    "crypto/md5"
    "encoding/hex"
    "net/url"
    "net/http"
    "log"
    "io/ioutil"
    "strings"
)

func TrimUrl(url string) string {
    res := strings.TrimSpace(url) 
    return strings.TrimSuffix("/")
}

func GetRoomId(url string) string {
    return strings.Splist(url, "/")[-1]
}

// generate md5 for roomMap key
func GenRoomKey(roomUrl string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(roomUrl)
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

// http GET
func HttpGet(url string, params map[string]string) string, error {
    u, _ := url.Parse(url)
    if len(params) > 0 {
        q := u.Query()
        for k, v := range params {
            q.Set(k, v)
        }
        u.RawQuery = q.Encode()
    } 
    res, err := http.Get(u.String())
    if err != nil {
        return nil, err
    }

    result, err := ioutil.ReadAll(res.Body)
    res.Body.Close()
    if err != nil {
        return nil, err
    }

    return result, nil
}
