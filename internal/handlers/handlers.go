package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/dustin/go-humanize"
	"github.com/gorilla/websocket"
)

var wsChan = make(chan WsPayload)
var clients = make(map[WebSocketConnection]*UserInfo)

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./html"),
	jet.InDevelopmentMode(),
)

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func Home(w http.ResponseWriter, r *http.Request) {
	err := renderPage(w, "home.jet", nil)
	if err != nil {
		log.Println(err)
	}
}

func Chat(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get(":name")
	if username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	log.Printf("User [%s] was login.\n", username)

	vars := make(jet.VarMap)
	vars.Set("username", username)
	err := renderPage(w, "chat.jet", vars)
	if err != nil {
		log.Println(err)
	}
}

type WebSocketConnection struct {
	*websocket.Conn
}

type UserInfo struct {
	Username           string    `json:"username"`
	Avatar             string    `json:"avatar"`
	ActiveHumanizeTime string    `json:"active_time"`
	ActiveTime         time.Time `json:"-"`
}

type WsJsonResponse struct {
	Action         string      `json:"action"`
	User           *UserInfo   `json:"user"`
	Message        string      `json:"message"`
	MessageType    string      `json:"message_type"`
	ConnectedUsers []*UserInfo `json:"connected_users"`
}

type WsPayload struct {
	Action   string              `json:"action"`
	UserName string              `json:"username"`
	Message  string              `json:"message"`
	Conn     WebSocketConnection `json:"-"`
}

func WSEndPoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	log.Println("Client connected to endpoint.")

	var response WsJsonResponse
	response.Message = "<em><small>Connected to server</small></em>"

	conn := WebSocketConnection{Conn: ws}
	clients[conn] = nil

	err = ws.WriteJSON(response)
	if err != nil {
		log.Println(err)
	}

	go ListenForWs(&conn)
}

func ListenForWs(conn *WebSocketConnection) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Error", fmt.Sprintf("%v", r))
		}
	}()

	var payload WsPayload
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			// no payload and do nothing
		} else {
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

func ListenToWsChannel() {
	var response WsJsonResponse

	for {
		e := <-wsChan

		switch e.Action {
		case "username":
			rand.Seed(time.Now().Unix())
			// get a list of all users and send it back via broadcase
			loginTime := time.Now()
			clients[e.Conn] = &UserInfo{
				Username:           e.UserName,
				Avatar:             fmt.Sprintf("avatar%d.png", rand.Intn(8)+1),
				ActiveHumanizeTime: humanize.Time(loginTime),
				ActiveTime:         loginTime,
			}
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			broadcastToAll(e.UserName, response)

		case "left":
			delete(clients, e.Conn)
			users := getUserList()
			response.Action = "list_users"
			response.ConnectedUsers = users
			broadcastToAll(e.UserName, response)

		case "broadcase":
			lastMessageSendTime := time.Now()

			response.Action = "broadcase"

			response.User = clients[e.Conn]
			response.User.ActiveHumanizeTime = humanize.Time(lastMessageSendTime)
			response.User.ActiveTime = lastMessageSendTime

			// response.Message = fmt.Sprintf("<strong>%s</strong>: %s", e.UserName, e.Message)
			response.Message = e.Message
			broadcastToAll(e.UserName, response)
		}
	}
}

func broadcastToAll(sendUserName string, response WsJsonResponse) {
	for conn, user := range clients {
		if user != nil {
			if sendUserName == user.Username {
				response.MessageType = "from"
			} else {
				response.MessageType = "to"
			}
		}

		err := conn.WriteJSON(response)
		if err != nil {
			log.Printf("websocket error, %s\n", err)
			_ = conn.Close()
			delete(clients, conn)
		}
	}
}

func getUserList() []*UserInfo {
	var userList []*UserInfo
	for _, x := range clients {
		if x != nil {
			userList = append(userList, x)
		}
	}
	// sort.Strings(userList)
	sort.Slice(userList, func(i, j int) bool {
		return userList[i].ActiveTime.Before(userList[j].ActiveTime)
	})
	return userList
}

func renderPage(w http.ResponseWriter, tmpl string, data jet.VarMap) error {
	view, err := views.GetTemplate(tmpl)
	if err != nil {
		return err
	}

	err = view.Execute(w, data, nil)
	if err != nil {
		return err
	}

	return nil
}
