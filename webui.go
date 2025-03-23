package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gopkg.in/telebot.v4"
)

type WebUI struct {
	addr     string
	user     string
	pass     string
	servemux *http.ServeMux
	tgBot    *TgBot
}

func NewWebUI(addr string, user string, pass string, tgBot *TgBot) (ui *WebUI) {
	ui = new(WebUI)
	ui.addr = addr
	ui.user = user
	ui.pass = pass
	ui.servemux = http.NewServeMux()
	ui.servemux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if ui.checkAuth(w, r) {
			http.ServeFile(w, r, "template/index.html")
		}
	})
	ui.servemux.HandleFunc("/bot/getUpdates", func(w http.ResponseWriter, r *http.Request) {
		if ui.checkAuth(w, r) {
			ui.getUpdatesHandler(w, r)
		}
	})
	ui.servemux.HandleFunc("/bot/sendMessage", func(w http.ResponseWriter, r *http.Request) {
		if ui.checkAuth(w, r) {
			ui.sendMessageHandler(w, r)
		}
	})
	ui.servemux.HandleFunc("/bot/sendChatAction", func(w http.ResponseWriter, r *http.Request) {
		if ui.checkAuth(w, r) {
			ui.sendChatActionHandler(w, r)
		}
	})
	ui.tgBot = tgBot
	return
}

func (ui *WebUI) ListenAndServe() error {
	return http.ListenAndServe(ui.addr, ui.servemux)
}

func (ui *WebUI) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	user, pass, _ := r.BasicAuth()
	if user != ui.user || pass != ui.pass {
		w.Header().Set("WWW-Authenticate", "Basic")
		http.Error(w, "Access Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func (ui *WebUI) getUpdatesHandler(w http.ResponseWriter, r *http.Request) {
	offset := r.FormValue("offset")
	lastIdx, err := strconv.ParseInt(offset, 0, 0)
	if err != nil {
		http.Error(w, "Invalid argument \"offset\"", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; encoding=UTF-8")
	w.WriteHeader(http.StatusOK)
	updates := ui.tgBot.GetUpdates(int(lastIdx))
	result := struct {
		Result *[]telebot.Update `json:"result"`
	}{
		Result: &updates,
	}
	stream, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	w.Write(stream)
}

func (ui *WebUI) sendMessageHandler(w http.ResponseWriter, r *http.Request) {
	text := r.FormValue("text")
	chat_id := r.FormValue("chat_id")
	reply_to_message_id := r.FormValue("reply_to_message_id")
	reply_to_message_id_int, _ := strconv.ParseInt(reply_to_message_id, 0, 0)
	send_options := []any{}
	if reply_to_message_id_int != 0 {
		send_options = []any{
			&telebot.SendOptions{
				ReplyTo: &telebot.Message{
					ID: int(reply_to_message_id_int),
				},
			},
		}
	}
	err := ui.tgBot.SendMessage(SimpleDestination{
		ID: chat_id,
	}, text, send_options...)
	if err != nil {
		http.Error(w, "Failed to send message", http.StatusServiceUnavailable)
		return
	}
	http.Error(w, "No Content", http.StatusNoContent)
}

func (ui *WebUI) sendChatActionHandler(w http.ResponseWriter, r *http.Request) {
	chat_id := r.FormValue("chat_id")
	action := r.FormValue("action")
	err := ui.tgBot.Bot.Notify(SimpleDestination{
		ID: chat_id,
	}, telebot.ChatAction(action))
	if err != nil {
		http.Error(w, "Failed to send chat action", http.StatusServiceUnavailable)
		return
	}
	http.Error(w, "No Content", http.StatusNoContent)
}
