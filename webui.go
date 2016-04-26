package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"github.com/tucnak/telebot"
)

type WebUI struct {
	addr        string
	user        string
	pass        string
	servemux    *http.ServeMux
	tgBot       *TgBot
}

func NewWebUI(addr string, user string, pass string, tgBot *TgBot) (self *WebUI) {
	self = new(WebUI)
	self.addr = addr
	self.user = user
	self.pass = pass
	self.servemux = http.NewServeMux()
	self.servemux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if self.checkAuth(w, r) {
			http.ServeFile(w, r, "template/index.html")
		}
	})
	self.servemux.HandleFunc("/bot/getUpdates", func(w http.ResponseWriter, r *http.Request) {
		if self.checkAuth(w, r) {
			self.getUpdatesHandler(w, r)
		}
	})
	self.servemux.HandleFunc("/bot/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if self.checkAuth(w, r) {
			self.sendMessageHandler(w, r)
		}
	})
	self.servemux.HandleFunc("/bot/sendChatAction", func(w http.ResponseWriter, r *http.Request) {
		if self.checkAuth(w, r) {
			self.sendChatActionHandler(w, r)
		}
	})
	self.tgBot = tgBot
	return
}

func (self *WebUI) ListenAndServe() error {
	return http.ListenAndServe(self.addr, self.servemux)
}

func (self *WebUI) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	user, pass, _ := r.BasicAuth()
	if user != self.user || pass != self.pass {
		w.Header().Set("WWW-Authenticate", "Basic")
		http.Error(w, "Access Unauthorized", 401)
		return false
	}
	return true
}

func (self *WebUI) getUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	offset := r.FormValue("offset")
	lastIdx, err := strconv.ParseInt(offset, 0, 0)
	if err != nil {
		http.Error(w, "Invalid argument \"offset\"", 400)
		return
	}
	w.Header().Set("Content-Type", "application/json; encoding=UTF-8")
	w.WriteHeader(200)
	updates := self.tgBot.GetUpdates(int(lastIdx))
	result := struct {
		Result *[]telebot.Update `json:"result"`
	} {
		Result: &updates,
	}
	stream, err := json.Marshal(result)
	if err != nil { panic(err) }
	w.Write(stream)
}

func (self *WebUI) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	text := r.FormValue("text")
	chat_id := r.FormValue("chat_id")
	reply_to_message_id := r.FormValue("reply_to_message_id")
	reply_to_message_id_int, _ := strconv.ParseInt(reply_to_message_id, 0, 0)
	var send_options *telebot.SendOptions
	if reply_to_message_id_int != 0 {
		send_options = new(telebot.SendOptions)
		send_options.ReplyTo.ID = int(reply_to_message_id_int)
	}
	err := self.tgBot.SendMessage(SimpleDestination {
		ID: chat_id,
	}, text, send_options)
	if err != nil {
		http.Error(w, "Failed to send message", 503)
		return
	}
	http.Error(w, "No Content", 204)
}

func (self *WebUI) sendChatActionHandler(w http.ResponseWriter, r *http.Request) {
	chat_id := r.FormValue("chat_id")
	action := r.FormValue("action")
	err := self.tgBot.Bot.SendChatAction(SimpleDestination {
		ID: chat_id,
	}, action)
	if err != nil {
		http.Error(w, "Failed to send chat action", 503)
		return
	}
	http.Error(w, "No Content", 204)
}
