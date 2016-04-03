package main

import (
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"github.com/tucnak/telebot"
)

type TgBot struct {
	Bot         *telebot.Bot
	mutex	    sync.RWMutex
	firstIdx    int
	messages    []telebot.Update
	queue       uintptr
	channel     chan uintptr
	botResp     *BotResp
}

func NewTgBot(secret string) (self *TgBot, err error) {
	self = new(TgBot)
	self.Bot, err = telebot.NewBot(secret)
	self.messages = make([]telebot.Update, 0, 512)
	self.channel = make(chan uintptr)
	self.botResp = NewBotResp(self)
	return
}

func (self *TgBot) addMessage(message *telebot.Message) {
	log.Printf("Message: %s\n", message.Text)
	defer func() {
		queue := atomic.SwapUintptr(&self.queue, 0)
		if queue != 0 {
			self.channel <- queue
		}
	}()
	self.mutex.Lock()
	defer self.mutex.Unlock()
	message_copy := *message
	if message.ReplyTo != nil {
		message_copy.ReplyTo = new(telebot.Message)
		*message_copy.ReplyTo = *message.ReplyTo
	}
	self.messages = append(self.messages, telebot.Update {
		ID: self.firstIdx + len(self.messages),
		Payload: message_copy,
	})
	if len(self.messages) >= 512 {
		oldmessages := self.messages
		self.messages = make([]telebot.Update, 256, 512)
		copy(self.messages, oldmessages[256:])
		self.firstIdx += 256
	}
}

func (self *TgBot) Run() {
	messages := make(chan telebot.Message)
	self.Bot.Listen(messages, 60*time.Second)

	for message := range messages {
		self.addMessage(&message)
		go func(ask string) {
			response := self.botResp.GetResponse(ask)
			if len(response) != 0 {
				self.SendMessage(message.Chat, response, nil)
			}
		}(message.Text)
	}
	return
}

func (self *TgBot) GetUpdates(lastIdx int) (messages []telebot.Update) {
	if lastIdx < self.firstIdx {
		lastIdx = self.firstIdx
	}
	for len(self.messages)+self.firstIdx-lastIdx <= 0 {
		atomic.AddUintptr(&self.queue, 1)
		queue := <-self.channel
		if queue != 1 {
			self.channel <- queue-1
		}
	}
	self.mutex.RLock()
	defer self.mutex.RUnlock()
	messages = make([]telebot.Update, len(self.messages)+self.firstIdx-lastIdx)
	copy(messages, self.messages[lastIdx-self.firstIdx:])
	return
}

func (self *TgBot) SendMessage(recipient telebot.Recipient, message string, options *telebot.SendOptions) (err error) {
	err = self.Bot.SendMessage(recipient, message, options)
	if err == nil {
		msg := new(telebot.Message)
		msg.Unixtime = int(time.Now().Unix())
		msg.Text = message
		msg.Sender.FirstName = "Me"
		if chat, ok := recipient.(telebot.Chat); ok {
			msg.Chat = chat
		} else if user, ok := recipient.(telebot.User); ok {
			msg.Chat.ID = int64(user.ID)
		} else {
			dest, _ := strconv.ParseInt(recipient.Destination(), 0, 0)
			msg.Chat.ID = dest
		}
		if options != nil {
			msg.ReplyTo = &options.ReplyTo
		}
		self.addMessage(msg)
	}
	return
}

type SimpleDestination struct {
	ID	        string
}

func (self SimpleDestination) Destination() string {
	return self.ID
}
