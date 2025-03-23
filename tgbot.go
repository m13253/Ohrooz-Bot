package main

import (
	"log"
	"sync"
	"sync/atomic"

	"gopkg.in/telebot.v4"
)

type TgBot struct {
	Bot      *telebot.Bot
	mutex    sync.RWMutex
	firstIdx int
	messages []telebot.Update
	queue    uintptr
	channel  chan uintptr
	botResp  *BotResp
}

func NewTgBot(secret string) (bot *TgBot, err error) {
	bot = new(TgBot)
	bot.Bot, err = telebot.NewBot(telebot.Settings{
		Token: secret,
	})
	bot.messages = make([]telebot.Update, 0, 512)
	bot.channel = make(chan uintptr)
	bot.botResp = NewBotResp(bot)
	return
}

func (bot *TgBot) addMessage(message *telebot.Message) {
	log.Printf("Message: %s\n", message.Text)
	defer func() {
		queue := atomic.SwapUintptr(&bot.queue, 0)
		if queue != 0 {
			bot.channel <- queue
		}
	}()
	bot.mutex.Lock()
	defer bot.mutex.Unlock()
	message_copy := *message
	if message.ReplyTo != nil {
		message_copy.ReplyTo = new(telebot.Message)
		*message_copy.ReplyTo = *message.ReplyTo
	}
	bot.messages = append(bot.messages, telebot.Update{
		ID:      bot.firstIdx + len(bot.messages),
		Message: &message_copy,
	})
	if len(bot.messages) >= 512 {
		oldmessages := bot.messages
		bot.messages = make([]telebot.Update, 256, 512)
		copy(bot.messages, oldmessages[256:])
		bot.firstIdx += 256
	}
}

func (bot *TgBot) Run() {
	go func() {
		bot.Bot.Poller.Poll(bot.Bot, bot.Bot.Updates, make(chan struct{}))
	}()
	for update := range bot.Bot.Updates {
		message := update.Message
		if message == nil {
			continue
		}
		bot.addMessage(message)
		go func(ask string) {
			response := bot.botResp.GetResponse(ask)
			if len(response) != 0 {
				bot.SendMessage(message.Chat, response, nil)
			}
		}(message.Text)
	}
}

func (bot *TgBot) GetUpdates(lastIdx int) (messages []telebot.Update) {
	if lastIdx < bot.firstIdx {
		lastIdx = bot.firstIdx
	}
	for len(bot.messages)+bot.firstIdx-lastIdx <= 0 {
		atomic.AddUintptr(&bot.queue, 1)
		queue := <-bot.channel
		if queue != 1 {
			bot.channel <- queue - 1
		}
	}
	bot.mutex.RLock()
	defer bot.mutex.RUnlock()
	messages = make([]telebot.Update, len(bot.messages)+bot.firstIdx-lastIdx)
	copy(messages, bot.messages[lastIdx-bot.firstIdx:])
	return
}

func (bot *TgBot) SendMessage(recipient telebot.Recipient, message string, options *telebot.SendOptions) (err error) {
	msg, err := bot.Bot.Send(recipient, message, options)
	if err != nil {
		return err
	}
	bot.addMessage(msg)
	return
}

type SimpleDestination struct {
	ID string
}

func (dest SimpleDestination) Recipient() string {
	return dest.ID
}
