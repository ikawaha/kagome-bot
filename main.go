package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: bot slack-bot-token\n")
		os.Exit(1)
	}

	// start a websocket-based Real Time API session
	bot, err := NewBot(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	fmt.Println("^C exits")

	for {
		msg, err := bot.GetMessage()
		if err != nil {
			log.Printf("receive error, %v", err)
			bot.Close()
			if bot, err = NewBot(os.Args[1]); err != nil { // reboot
				log.Fatalf("reboot failed, %v", err)
			}
			log.Printf("reboot")
			continue
		}
		log.Printf("bot_id: %v, msg_user_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
		if bot.ID != msg.MentionID() || msg.Type != "message" && msg.SubType != "" {
			continue
		}
		go bot.Response(msg)
	}
}
