package client

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func newMessageTokenizeHandlerFunc(ctx context.Context, botID string) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		eventPayload, ok := event.Data.(slackevents.EventsAPIEvent)
		if !ok {
			client.Debugf("skipped Envelope: %v", event)
			return
		}
		client.Ack(*event.Request)
		p, ok := eventPayload.InnerEvent.Data.(*slackevents.MessageEvent)
		if !ok {
			client.Debugf("skipped Payload Event: %v", event)
			return
		}
		if !strings.HasPrefix(p.Text, botID) {
			client.Debugf("skipped message")
			return
		}
		s := strings.TrimSpace(p.Text[len(botID):])
		if s == "" {
			ack(ctx, client, p.Channel)
			return
		}
		postTokenizeResult(ctx, client, s, p.Channel, ipaDict)
	}
}

func getDictType(cmd string) dictKind {
	dictType := ipaDict
	if strings.HasSuffix(cmd, string(neologdDict)) {
		dictType = neologdDict
	} else if strings.HasSuffix(cmd, string(uniDict)) {
		dictType = uniDict
	}
	return dictType
}

func newSlashCommandTokenizeHandlerFunc(ctx context.Context) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		ev, ok := event.Data.(slack.SlashCommand)
		if !ok {
			client.Debugf("skipped command: %v", event)
		}
		client.Ack(*event.Request)
		dict := getDictType(ev.Command)
		cmd := fmt.Sprintf("%v %v", ev.Command, ev.Text)
		if _, _, err := client.PostMessage(ev.ChannelID, slack.MsgOptionText(cmd, false)); err != nil {
			client.Debugf("failed to post message: %v", err)
			return
		}
		if ev.Text == "" {
			ack(ctx, client, ev.ChannelID)
			return
		}
		postTokenizeResult(ctx, client, ev.Text, ev.ChannelID, dict)
	}
}

func defaultHandler(event *socketmode.Event, client *socketmode.Client) {
	// fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
	client.Debugf("skip event: %v", event.Type)
}

func ack(ctx context.Context, client *socketmode.Client, channel string) {
	bver := BotVersionFromContext(ctx)
	tver := TokenizerVersionFromContext(ctx)
	msg := fmt.Sprintf("呼んだ？ (bot: %s/ kagome: %s)", bver, tver)
	if _, _, err := client.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
		log.Printf("ack failed: %v", err)
	}
}

func postTokenizeResult(ctx context.Context, client *socketmode.Client, txt string, channel string, dict dictKind) {
	resp, err := tokenize(ctx, txt, dict)
	if err != nil {
		log.Printf("create lattice image error, %v", err)
		msg := fmt.Sprintf("形態素解析に失敗しちゃいました．%q です", err)
		if _, _, err := client.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
			log.Printf("post message failed, msg: %+v, %v", txt, err)
		}
		return
	}
	_, tt, err := client.PostMessage(channel, slack.MsgOptionText(resp.comment, false))
	if err != nil {
		log.Printf("post message failed, msg: %s, %v", txt, err)
		return
	}
	if _, err = client.UploadFileV2Context(ctx, slack.UploadFileV2Parameters{
		File:            "lattice.png",
		FileSize:        resp.image.Len(),
		Reader:          resp.image,
		Filename:        UploadImageFileName,
		Title:           resp.title,
		Channel:         channel,
		ThreadTimestamp: tt,
		AltTxt:          txt,
		SnippetText:     "kagome-bot",
	}); err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}
