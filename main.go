package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ikawaha/kagome-bot/client"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: client app-level-token slack-client-token\n")
		os.Exit(1)
	}
	var debug bool
	bot, err := client.New(os.Args[1], os.Args[2], debug)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("^C exits")
	if err = bot.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
