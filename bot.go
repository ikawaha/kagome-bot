package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome-bot/slack"
	"github.com/ikawaha/kagome/tokenizer"
)

const (
	graphvizCmd = "circo"
	cmdTimeout  = 25 * time.Second
)

type Bot struct {
	*slack.Client
}

func NewBot(token string) (*Bot, error) {
	c, err := slack.New(token)
	if err != nil {
		return nil, err
	}
	return &Bot{Client: c}, err
}

func (bot Bot) Response(msg slack.Message) {
	sen := msg.TextBody()
	if len(sen) == 0 {
		msg.Text = "呼んだ？"
		bot.PostMessage(msg)
		return
	}
	img, tokens, err := createTokenizeLatticeImage(sen)
	if err != nil {
		log.Printf("create lattice image error, %v", err)
		msg.Text = fmt.Sprintf("形態素解析に失敗しちゃいました．%v です", err)
		bot.PostMessage(msg)
		return
	}
	comment := "```" + yield(tokens) + "```"
	if err := bot.UploadImage(msg.Channel, sen, "lattice.png", "png", comment, img); err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}

func createTokenizeLatticeImage(sen string) (io.Reader, []tokenizer.Token, error) {
	if _, err := exec.LookPath(graphvizCmd); err != nil {
		return nil, nil, fmt.Errorf("circo/graphviz is not installed in your $PATH")
	}
	var buf bytes.Buffer
	cmd := exec.Command("dot", "-Tpng")
	r0, w0 := io.Pipe()
	cmd.Stdin = r0
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("process done with error = %v", err)
	}
	t := tokenizer.New()
	tokens := t.AnalyzeGraph(sen, tokenizer.Normal, w0)
	w0.Close()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(cmdTimeout):
		if err := cmd.Process.Kill(); err != nil {
			return nil, nil, fmt.Errorf("failed to kill, %v", err)
		}
		<-done
		return nil, nil, fmt.Errorf("graphviz timeout")
	case err := <-done:
		if err != nil {
			return nil, nil, fmt.Errorf("process done with error, %v", err)
		}
	}
	return &buf, tokens, nil
}

func yield(tokens []tokenizer.Token) string {
	var buf bytes.Buffer
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		fmt.Fprintf(&buf, "%s\t%s\n", token.Surface, strings.Join(token.Features(), ","))
	}
	return buf.String()
}
