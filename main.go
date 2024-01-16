package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ikawaha/slackbot/socketmode"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "usage: bot app-level-token slack-bot-token name")
		os.Exit(1)
	}

	// start a websocket-based Real Time API session
	bot, err := NewBot(os.Args[1], os.Args[2], os.Args[3])
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Close()
	fmt.Println("^C exits")

	mentionTag := "<@" + bot.ID + ">"
	for {
		if err := bot.ReceiveMessage(context.TODO(), func(ctx context.Context, ev *socketmode.Event) error {
			log.Printf("bot_id: %v, msg_user_id: %v, event: %+v\n", bot.ID, ev.UserID, ev)
			if ev.IsSlashCommand() {
				go bot.Response(ev)
				return nil
			}
			if ev.IsMessage() && strings.HasPrefix(ev.Text, mentionTag) {
				ev.Text = strings.TrimSpace(ev.Text[len(mentionTag):])
				go bot.Response(ev)
				return nil
			}
			return nil
		}); err != nil {
			log.Fatal(err)
		}
	}
}
