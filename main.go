package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ikawaha/kagome-bot/client"
	"runtime/debug"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: client app-level-token slack-client-token\n")
		os.Exit(1)
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Fatal("Failed to read build info")
	}
	var versions client.Versions
	for _, dep := range bi.Deps {
		if strings.Contains(dep.Path, "ikawaha/kagome/v2") {
			versions = client.Versions{
				Kagome: dep.Version,
				Bot:    bi.Main.Version,
			}
		}
	}
	var dflag bool
	bot, err := client.New(os.Args[1], os.Args[2], dflag, versions)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("^C exits")
	if err = bot.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}
