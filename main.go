package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: bot app-level-token slack-bot-token")
		os.Exit(1)
	}
	debug := false
	bot, err := NewBot(os.Args[1], os.Args[2], debug)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	fmt.Println("^C exits")

	err = bot.Run()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
}
