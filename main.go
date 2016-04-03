package main

import (
	"log"
)

func main() {
	tgBot, err := NewTgBot(SECRET)
	if err != nil { log.Fatalln(err) }
	go tgBot.Run()
	webUI := NewWebUI(LISTEN, USER, PASS, tgBot)
	err = webUI.ListenAndServe()
	if err != nil { log.Fatalln(err) }
}
