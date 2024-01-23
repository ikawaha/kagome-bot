package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome-dict/uni"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome-dict-ipa-neologd"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/slack-go/slack/socketmode"
)

const (
	GraphvizCmd = "circo"
	CmdTimeout  = 25 * time.Second

	UploadFileType      = "png"
	UploadImageFileName = "lattice.png"
)

type Bot struct {
	c  *socketmode.Client
	id string
}

func NewBot(appToken, botToken string, debug bool) (*Bot, error) {
	webApi := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionDebug(debug),
	)
	bot := socketmode.New(
		webApi,
		socketmode.OptionDebug(debug),
	)
	authTest, authTestErr := webApi.AuthTest()
	if authTestErr != nil {
		return nil, fmt.Errorf("slack-bot-token is invalid: %v\n", authTestErr)
	}
	botUserIdPrefix := fmt.Sprintf("<@%v>", authTest.UserID)
	return &Bot{c: bot, id: botUserIdPrefix}, nil
}

func (bot Bot) Run() error {
	socketModeHandler := socketmode.NewSocketmodeHandler(bot.c)
	socketModeHandler.HandleEvents(slackevents.AppMention, func(event *socketmode.Event, client *socketmode.Client) {
		tokenizeForAppMention(event, client, bot.id)
	})
	socketModeHandler.HandleEvents(slackevents.Message, func(event *socketmode.Event, client *socketmode.Client) {
		tokenizeForMessage(event, client, bot.id)
	})
	socketModeHandler.Handle(socketmode.EventTypeSlashCommand, tokenizeForSlashCommand)
	socketModeHandler.HandleDefault(defaultHandler)

	return socketModeHandler.RunEventLoop()
}

func defaultHandler(event *socketmode.Event, client *socketmode.Client) {
	// fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
	client.Debugf("Skip event: %v", event.Type)
}

func tokenizeForMessage(event *socketmode.Event, client *socketmode.Client, mentionId string) {
	eventPayload, ok := event.Data.(slackevents.EventsAPIEvent)
	if !ok {
		client.Debugf("Skipped Envelope: %v", event)
		return
	}
	client.Ack(*event.Request)
	payloadEvent, ok := eventPayload.InnerEvent.Data.(*slackevents.MessageEvent)
	if !ok {
		client.Debugf("Skipped Payload Event: %v", event)
		return
	}
	if strings.HasPrefix(payloadEvent.Text, mentionId) {
		sentence := strings.TrimSpace(payloadEvent.Text[len(mentionId):])
		response(client, sentence, payloadEvent.Channel)
	} else {
		client.Debugf("Skipped message")
		return
	}
}

func tokenizeForAppMention(event *socketmode.Event, client *socketmode.Client, mentionId string) {
	eventPayload, ok := event.Data.(slackevents.EventsAPIEvent)
	if !ok {
		client.Debugf("Skipped Envelope: %v", event)
		return
	}
	client.Ack(*event.Request)
	appMentionEvent, ok := eventPayload.InnerEvent.Data.(*slackevents.AppMentionEvent)
	if !ok {
		client.Debugf("Skipped Payload Event: %v", event)
		return
	}
	sentence := strings.TrimSpace(appMentionEvent.Text[len(mentionId):])
	response(client, sentence, appMentionEvent.Channel)
}

func tokenizeForSlashCommand(event *socketmode.Event, client *socketmode.Client) {
	ev, ok := event.Data.(slack.SlashCommand)
	if !ok {
		client.Debugf("Skipped command: %v", event)
	}
	client.Ack(*event.Request)
	dictType := IPA
	if strings.HasSuffix(ev.Command, string(NEOLOGD)) {
		dictType = NEOLOGD
	} else if strings.HasSuffix(ev.Command, string(UNI)) {
		dictType = UNI
	}
	command := fmt.Sprintf("%v %v", ev.Command, ev.Text)
	_, _, err := client.PostMessage(ev.ChannelID, slack.MsgOptionText(command, false))
	if err != nil {

	}
	responseWithDictType(client, ev.Text, ev.ChannelID, dictType)
}

func response(c *socketmode.Client, sentence string, channel string) {
	responseWithDictType(c, sentence, channel, IPA)
}

func responseWithDictType(c *socketmode.Client, sen string, channel string, dictionary DictType) {
	if len(sen) == 0 {
		msg := "呼んだ？"
		if _, _, err := c.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
			log.Printf("post message failed, msg: %+v, %v", sen, err)
		}
		return
	}
	img, tokens, err := createTokenizeLatticeImage(sen, dictionary)
	if err != nil {
		log.Printf("create lattice image error, %v", err)
		msg := fmt.Sprintf("形態素解析に失敗しちゃいました．%v です", err)
		if _, _, err := c.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
			log.Printf("post message failed, msg: %+v, %v", sen, err)
		}
		return
	}
	comment := "```" + yield(tokens) + "```"
	_, err = c.UploadFile(
		slack.FileUploadParameters{
			Reader:         img,
			Filename:       UploadImageFileName,
			Channels:       []string{channel},
			Filetype:       UploadFileType,
			InitialComment: comment,
			Title:          sen,
		})
	if err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}

func createTokenizeLatticeImage(sen string, dictType DictType) (io.Reader, []tokenizer.Token, error) {
	var dictionary *dict.Dict
	if dictType == UNI {
		dictionary = uni.Dict()
	} else if dictType == NEOLOGD {
		dictionary = ipaneologd.Dict()
	} else {
		dictionary = ipa.Dict()
	}
	t, err := tokenizer.New(dictionary)
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

type DictType string

const (
	IPA     DictType = "ipa"
	UNI     DictType = "uni"
	NEOLOGD DictType = "neologd"
)
