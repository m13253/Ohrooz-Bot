package main

import (
	"math/rand"
	"regexp"
	"time"
)

type BotResp struct {
	tgBot       *TgBot
	random      *rand.Rand
}

func NewBotResp(tgBot *TgBot) (self *BotResp) {
	self = new(BotResp)
	self.tgBot = tgBot
	self.random = rand.New(rand.NewSource(time.Now().UnixNano()))
	return
}

func (self *BotResp) GetResponse(ask string) (resp string) {
	if ok, _ := regexp.MatchString("^/?test(@|$)", ask); ok {
		return "test ok"
	}
	if ok, _ := regexp.MatchString("^/?ping(@|$)", ask); ok {
		return "pong"
	}
	if ok, _ := regexp.MatchString("^/help(@|$)", ask); ok {
		return "喵呜！"
	}
	if ok, _ := regexp.MatchString("^/start(@|$)", ask); ok {
		return "喵呜？"
	}
	return
}
