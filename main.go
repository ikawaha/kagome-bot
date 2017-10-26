package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ikawaha/kagome-bot/client"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: bot slack-bot-token\n")
		os.Exit(1)
	}

	// start a websocket-based Real Time API session
	bot, err := client.New(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	fmt.Println("^C exits")

	for {
		msg, err := bot.GetMessage()
		if err != nil {
			log.Printf("receive error, %v", err)
		}
		log.Printf("bot_id: %v, msg_user_id: %v, msg:%+v\n", bot.ID, msg.UserID, msg)
		if bot.ID != msg.MentionID() || msg.Type != "message" && msg.SubType != "" {
			continue
		}
		go func(m client.Message) {
			sen := m.TextBody()
			if len(sen) == 0 {
				m.Text = "呼んだ？"
				bot.PostMessage(m)
				return
			}
			img, tokens, err := createTokenizeLatticeImage(sen)
			if err != nil {
				log.Printf("create lattice image error, %v", err)
				m.Text = fmt.Sprintf("形態素解析に失敗しちゃいました．%v です", err)
				bot.PostMessage(m)
				return
			}
			comment := "```" + yield(tokens) + "```"
			if err := bot.UploadImage(m.Channel, sen, "lattice.png", "png", comment, img); err != nil {
				log.Printf("upload lattice image error, %v", err)
			}
		}(msg)
	}
}
