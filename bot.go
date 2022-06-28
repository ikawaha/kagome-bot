package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
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

func NewBot(appToken, botToken, botName string) (*Bot, error) {
	c, err := slackbot.New(appToken, botToken, slackbot.SetBotID(botName))
	if err != nil {
		return nil, err
	}
	return &Bot{Client: c}, err
}

func (bot Bot) Response(e *slackbot.Event) {
	sen := e.Text
	if len(sen) == 0 {
		e.Text = "呼んだ？"
		if err := bot.PostMessage(context.TODO(), e.Channel, e.Text); err != nil {
			log.Printf("post message failed, msg: %+v, %v", e, err)
		}
		return
	}
	img, tokens, err := createTokenizeLatticeImage(sen)
	if err != nil {
		log.Printf("create lattice image error, %v", err)
		e.Text = fmt.Sprintf("形態素解析に失敗しちゃいました．%v です", err)
		if err := bot.PostMessage(context.TODO(), e.Channel, e.Text); err != nil {
			log.Printf("post message failed, msg: %+v, %v", e, err)
		}
		return
	}
	comment := "```" + yield(tokens) + "```"
	if err := bot.UploadImage(context.TODO(), []string{e.Channel}, sen, UploadImageFileName, UploadFileType, comment, img); err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}

func createTokenizeLatticeImage(sen string) (io.Reader, []tokenizer.Token, error) {
	t, err := tokenizer.New(ipa.Dict())
	if err != nil {
		return nil, nil, fmt.Errorf("tokenizer initialization failed, %w", err)
	}
	if _, err := exec.LookPath(GraphvizCmd); err != nil {
		return nil, nil, fmt.Errorf("command %v is not installed in your $PATH", GraphvizCmd)
	}
	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), CmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "dot", "-T"+UploadFileType)
	r0, w0 := io.Pipe()
	cmd.Stdin = r0
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("process done with error, %w", err)
	}
	tokens := t.AnalyzeGraph(w0, sen, tokenizer.Normal)
	if err := w0.Close(); err != nil {
		return nil, nil, fmt.Errorf("pipe close error, %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("process done with error, %w", err)
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
