package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome/tokenizer"
	"github.com/ikawaha/slackbot"
)

const (
	GraphvizCmd = "circo"
	CmdTimeout  = 25 * time.Second

	UploadFileType      = "png"
	UploadImageFileName = "lattice.png"
)

type Bot struct {
	*slackbot.Client
}

func NewBot(token string) (*Bot, error) {
	c, err := slackbot.New(token)
	if err != nil {
		return nil, err
	}
	return &Bot{Client: c}, err
}

func (bot Bot) Response(msg slackbot.Message) {
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
	if err := bot.UploadImage([]string{msg.Channel}, sen, UploadImageFileName, UploadFileType, comment, img); err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}

func createTokenizeLatticeImage(sen string) (io.Reader, []tokenizer.Token, error) {
	if _, err := exec.LookPath(GraphvizCmd); err != nil {
		return nil, nil, fmt.Errorf("command %v is not installed in your $PATH", GraphvizCmd)
	}
	var buf bytes.Buffer
	cmd := exec.Command("dot", "-T"+UploadFileType)
	r0, w0 := io.Pipe()
	cmd.Stdin = r0
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("process done with error = %v", err)
	}
	t := tokenizer.New()
	tokens := t.AnalyzeGraph(w0, sen, tokenizer.Normal)
	w0.Close()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(CmdTimeout):
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
