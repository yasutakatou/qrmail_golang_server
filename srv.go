package main

import (
  "image/png"
  "github.com/boombuler/barcode"
  "github.com/boombuler/barcode/qr"
  "golang.org/x/net/context"
  "golang.org/x/oauth2"
  "golang.org/x/oauth2/google"
  "google.golang.org/api/gmail/v1"
  "encoding/base64"
  "io/ioutil"
  "strings"
  "github.com/garyburd/redigo/redis"
  "fmt"
  "os"
  "net/http"
  "encoding/json"
  "log"
  "net/url"
  "os/user"
  "path/filepath"
  "math/rand"
  "time"
  "bytes"
  "flag"
  "io"

  _ "reflect"
)

type reqData struct {
  Command string `json:"command"`
  Params  string `json:"params"`
}

func redisSet(key string, value string, c redis.Conn){
  c.Do("SET", key, value)
}

func redisDel(key string, c redis.Conn){
  c.Do("DEL", key)
}

func redisSetList(key string, value []string, c redis.Conn){
  for _ , v := range value {
    fmt.Println(v)
    c.Do("RPUSH", key, v)
  }
}

func redisGet(key string, c redis.Conn) string{
  s, err := redis.String(c.Do("GET", key))
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  return s
}

func init() {
  rand.Seed(time.Now().UnixNano())
}

func redisGetList(key string, c redis.Conn) []string{
  s, err := redis.Strings(c.Do("LRANGE", key, 0, -1))
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  return s
}

func redisConnection(redisIP string) redis.Conn {
  c, err := redis.Dial("tcp", redisIP)
  if err != nil {
    panic(err)
  }
  return c
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
  cacheFile, err := tokenCacheFile()
  if err != nil {
      log.Fatalf("Unable to get path to cached credential file. %v", err)
  }
  tok, err := tokenFromFile(cacheFile)
  if err != nil {
      tok = getTokenFromWeb(config)
      saveToken(cacheFile, tok)
  }
  return config.Client(ctx, tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
  authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
  fmt.Printf("Go to the following link in your browser then type the "+
      "authorization code: \n%v\n", authURL)

  var code string
  if _, err := fmt.Scan(&code); err != nil {
      log.Fatalf("Unable to read authorization code %v", err)
  }

  tok, err := config.Exchange(oauth2.NoContext, code)
  if err != nil {
      log.Fatalf("Unable to retrieve token from web %v", err)
  }
  return tok
}

func tokenCacheFile() (string, error) {
  usr, err := user.Current()
  if err != nil {
      return "", err
  }
  tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
  os.MkdirAll(tokenCacheDir, 0700)
  return filepath.Join(tokenCacheDir,
      url.QueryEscape("gmail-go-quickstart.json")), err
}

func tokenFromFile(file string) (*oauth2.Token, error) {
  f, err := os.Open(file)
  if err != nil {
      return nil, err
  }
  t := &oauth2.Token{}
  err = json.NewDecoder(f).Decode(t)
  defer f.Close()
  return t, err
}

func saveToken(file string, token *oauth2.Token) {
  fmt.Printf("Saving credential file to: %s\n", file)
  f, err := os.Create(file)
  if err != nil {
      log.Fatalf("Unable to cache oauth token: %v", err)
  }
  defer f.Close()
  json.NewEncoder(f).Encode(token)
}

var rs1Letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStr(n int) string {
  b := make([]rune, n)
  for i := range b {
      b[i] = rs1Letters[rand.Intn(len(rs1Letters))]
  }
  return string(b)
}

type sendData struct {
  From    string `json:"from"`
  To      string `json:"to"`
  Title string `json:"title"`
  Bodys    string `json:"bodys"`
  File    string `json:"file"`
  Flag    string `json:"flag"`
}

type readData struct {
  Token    string `json:"token"`
}

type authData struct {
  Mail  string `json:"Mail"`
  IMEI  string `json:"Imei"`
  Token string `json:"Token"`
}

type fileData struct {
  Token string `json:"token"`
  Name  string `json:"name"`
}

func main() {
  _apisrver     := flag.String("api","127.0.0.1:8887","[-api=API Server and Port]")
  _dbsrver     := flag.String("db","127.0.0.1","[-db=DB Server and Port]")
  _redisServer := flag.String("redis","127.0.0.1:6379","[-redis=REDIS Server and Port]")
  
  flag.Parse()

  apiServer    := string(*_apisrver)  
  dbServer    := string(*_dbsrver)
  redisServer := string(*_redisServer)
  
  fmt.Println(" --------")
  fmt.Println(" |api   | ",apiServer)
  fmt.Println(" |db    | ",dbServer)
  fmt.Println(" |redis | ",redisServer)
  fmt.Println(" --------")
  
  http.HandleFunc("/send", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    d := json.NewDecoder(r.Body)
    p := &sendData{}
    err := d.Decode(p)
    if err != nil {
      fmt.Fprintln(w, "Error: internal server error")
      return
    }
    fmt.Println("From: " + p.From + " To: " + p.To + " Title: " + p.Title + " Bodys: " + p.Bodys + " File:" + p.File + " Flag: "+ p.Flag)

    randstring := randStr(4)

    c := redisConnection(redisServer)
    defer c.Close()

    var vallist = []string{p.From, p.To, p.Title, p.Bodys, p.File ,p.Flag}
    redisSetList(randstring, vallist, c)
 
    ctx := context.Background()

    b, err := ioutil.ReadFile("credentials.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, err := google.ConfigFromJSON(b, gmail.MailGoogleComScope)
    if err != nil {
        log.Fatalf("Unable to parse client secret file to config: %v", err)
    }
    client := getClient(ctx, config)

    srv, err := gmail.New(client)
    if err != nil {
        log.Fatalf("Unable to retrieve gmail Client %v", err)
    }

    var message gmail.Message

    bodyString := "https://" + dbServer + "/view?token=" + randstring

    temp := []byte("From: " + p.From + "\r\n" +
        "reply-to: " + p.To + "\r\n" +
        "To: " + p.To + "\n" +
        "Subject: " + p.Title + "\r\n" +
        "\r\n" + bodyString)

    message.Raw = base64.StdEncoding.EncodeToString(temp)
    message.Raw = strings.Replace(message.Raw, "/", "_", -1)
    message.Raw = strings.Replace(message.Raw, "+", "-", -1)
    message.Raw = strings.Replace(message.Raw, "=", "", -1)

    _, err = srv.Users.Messages.Send("me", &message).Do()
    if err != nil {
        log.Fatalf("Unable to send. %v", err)
    }

    /* delete mail. */

    qrCode, _ := qr.Encode("https://" + apiServer + "/del?token=" + randstring, qr.M, qr.Auto)
    qrCode, _ = barcode.Scale(qrCode, 200, 200) 

    randstring = randStr(4)  + ".png"
    file, _ := os.Create(randstring)
    defer file.Close()
    png.Encode(file, qrCode)

    fileBytes, err := ioutil.ReadFile(randstring)
    if err != nil {
      log.Fatalf("Unable to read file for attachment: %v", err)
    }
    fileData := base64.StdEncoding.EncodeToString(fileBytes) 

    ctx = context.Background()

    b, err = ioutil.ReadFile("credentials.json")
    if err != nil {
        log.Fatalf("Unable to read client secret file: %v", err)
    }

    config, err = google.ConfigFromJSON(b, gmail.MailGoogleComScope)
    if err != nil {
        log.Fatalf("Unable to parse client secret file to config: %v", err)
    }
    client = getClient(ctx, config)

    srv, err = gmail.New(client)
    if err != nil {
        log.Fatalf("Unable to retrieve gmail Client %v", err)
    }

    bodyString = "<html><body><img src=\"data:image/png;base64," + fileData + "\" width=\"200\" height=\"200\"></body></html>"
    mime := "MIME-version: 1.0\r\nContent-Type: text/html\r\n";
  
    temp = []byte("From: no-reply@anonymous.com\r\n" +
      "reply-to: " + p.From + "\r\n" +
      "To: " + p.From + "\n" +
      "Subject: [DELETE] " + p.Title + "\r\n" +
      mime + "\r\n" + 
      bodyString)

    message.Raw = base64.StdEncoding.EncodeToString(temp)
    message.Raw = strings.Replace(message.Raw, "/", "_", -1)
    message.Raw = strings.Replace(message.Raw, "+", "-", -1)
    message.Raw = strings.Replace(message.Raw, "=", "", -1)

    _, err = srv.Users.Messages.Send("me", &message).Do()
    if err != nil {
        log.Fatalf("Unable to send. %v", err)
    }

    //os.Remove(randstring)
  })

  http.HandleFunc("/read", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    d := json.NewDecoder(r.Body)
    p := &readData{}
    err := d.Decode(p)
    if err != nil {
      fmt.Fprintln(w, "Error: internal server error")
      return
    }
    fmt.Println("read: " + p.Token)

    c := redisConnection(redisServer)
    defer c.Close()

    sl := redisGetList(p.Token, c)

    if sl[5] == "0" { 
      fmt.Println("no read flag")
      datas := sendData{
        From: "",
        To: "",
        Title: "",
        Bodys: "",
        File: "",
        Flag: "",
      }
      jsonMsg, _ := json.Marshal(datas)
      fmt.Fprintln(w, string(jsonMsg))  
      return
    }

    datas := sendData{
      From: sl[0],
      To: sl[1],
      Title: sl[2],
      Bodys: sl[3],
      File: sl[4],
      Flag: sl[5],
    }
    jsonMsg, _ := json.Marshal(datas)
    fmt.Fprintln(w, string(jsonMsg))

    //mail delete after read.
    //redisDel(p.Token, c)
  })

  http.HandleFunc("/auth", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    d := json.NewDecoder(r.Body)
    p := &authData{}
    err := d.Decode(p)
    if err != nil {
      fmt.Fprintln(w, "Error: internal server error")
      return
    }
    fmt.Println("Mail: " + p.Mail + " IMEI: " + p.IMEI + " Token: " + p.Token)

    c := redisConnection(redisServer)
    defer c.Close()

    sl := redisGetList(p.Token, c)

    fmt.Println("From: " + sl[0] + " To: " + sl[1] + " Title: " + sl[2] + " Bodys: " + sl[3] + " File: "+ sl[4] + " Flag: "+ sl[5])

    if sl[1] != p.Mail {
      fmt.Println("address mismatch")
      return 
    }

    if sl[5] != "0" {
      fmt.Println("authorized")
      return 
    }

    if !sendHLauth("http://localhost:38080/query",p.Mail,p.IMEI) {
      fmt.Println("Hyperledger auth fail.")
      return       
    }

    redisDel(p.Token, c)

    datas := sendData{
      From: sl[0],
      To: sl[1],
      Title: sl[2],
      Bodys: sl[3],
      File: sl[4],
      Flag: "1",
    }

    var vallist = []string{datas.From, datas.To, datas.Title, datas.Bodys, datas.File, datas.Flag}
    redisSetList(p.Token, vallist, c)

    fmt.Println("auth success")
  })

  http.HandleFunc("/del", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    d := json.NewDecoder(r.Body)
    p := &authData{}
    err := d.Decode(p)
    if err != nil {
      fmt.Fprintln(w, "Error: internal server error")
      return
    }
    fmt.Println("Mail: " + p.Mail + " IMEI: " + p.IMEI + " Token: " + p.Token)

    c := redisConnection(redisServer)
    defer c.Close()

    sl := redisGetList(p.Token, c)

    fmt.Println("From: " + sl[0] + " To: " + sl[1] + " Title: " + sl[2] + " Bodys: " + sl[3] + " File: "+ sl[4] + " Flag: "+ sl[5])

    if sl[0] != p.Mail {
      fmt.Fprintln(w, "address mismatch")
      return 
    }

    if !sendHLauth("http://localhost:38080/query",p.Mail,p.IMEI) {
      fmt.Fprintln(w, "Hyperledger auth fail.")
      return       
    }

    redisDel(p.Token, c)

    datas := sendData{
      From: sl[0],
      To: sl[1],
      Title: sl[2],
      Bodys: "[DELETED]",
      File: "[DELETED]",
      Flag: "1",
    }

    var vallist = []string{datas.From, datas.To, datas.Title, datas.Bodys, datas.File, datas.Flag}
    redisSetList(p.Token, vallist, c)

    fmt.Fprintln(w, "del success")
  })

  http.HandleFunc("/download", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    if r.Method != "GET" {
      fmt.Fprintln(w, "not allow get request")
    }

    token := r.URL.Query()["token"][0]
    name := r.URL.Query()["name"][0]

    fmt.Println(" downloadToken: " + token)

    ff, err := os.Open("./" + token + "/" + name)
    if err != nil {
      fmt.Fprintln(w, "file open fail")
      return
    }

    raw, _ := ioutil.ReadAll(ff)
    w.WriteHeader(200)
    w.Header().Set("Content-Type", "application/octet-stream")
    w.Write(raw)
    return
  })

  http.HandleFunc("/upload", func (w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

    randstring := randStr(4)

    name := r.FormValue("name")
    file, _, err := r.FormFile("file")
    defer file.Close()

    if err != nil {
      fmt.Fprint(w, "upload error")
      return
    } 
    
    if err := os.Mkdir("./" + randstring, 0777); err != nil {
      fmt.Fprint(w, "make directory error")
      return
    }

    f, err := os.OpenFile("./" + randstring + "/" + name, os.O_WRONLY|os.O_CREATE, 0644)
    defer f.Close()
    if err != nil {
      fmt.Fprint(w, "file create error")
    } else {
      io.Copy(f, file)
    }
    fmt.Fprint(w, randstring)
  })

  err := http.ListenAndServeTLS(":28080", "./localhost.pem", "./localhost-key.pem", nil)
  //err := http.ListenAndServe(":28080", nil)
  if err != nil {
    log.Fatal("ListenAndServe: ", err)
  }
}

type hlData struct {
  Mail  string `json:"Mail"`
  IMEI  string `json:"IMEI"`
}

func sendHLauth(endpoint,mail,imei string) (bool) {
  data := hlData{
    Mail: mail,
    IMEI: imei,
  }

  payloadBytes, err := json.Marshal(data)
  if err != nil {
    fmt.Println(err)
    return false
  }

  client := &http.Client{}

  resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(payloadBytes))
  if err != nil {
    fmt.Println(err)
    return false
  }


  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Println(err)
    return false
  }

  if string(body) == "0" {
    return true
  }

  return false
}