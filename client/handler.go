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

func newMessageTokenizeHandlerFunc(ctx context.Context, botID string, info botInfo) socketmode.SocketmodeHandlerFunc {
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
		response(ctx, client, s, p.Channel, ipaDict, info)
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

func newSlashCommandTokenizeHandlerFunc(ctx context.Context, info botInfo) socketmode.SocketmodeHandlerFunc {
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
		response(ctx, client, ev.Text, ev.ChannelID, dict, info)
	}
}

func defaultHandler(event *socketmode.Event, client *socketmode.Client) {
	// fmt.Fprintf(os.Stderr, "Unexpected event type received: %s\n", evt.Type)
	client.Debugf("skip event: %v", event.Type)
}

func response(ctx context.Context, client *socketmode.Client, txt string, channel string, dict dictKind, info botInfo) {
	if len(txt) == 0 {
		msg := fmt.Sprintf("呼んだ？ (bot: %s/ kagome: %s)", info.botVersion, info.tokenizerVersion)
		if _, _, err := client.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
			log.Printf("post message failed, msg: %+v, %v", txt, err)
		}
		return
	}
	resp, err := tokenize(ctx, txt, dict)
	if err != nil {
		log.Printf("create lattice image error, %v", err)
		msg := fmt.Sprintf("形態素解析に失敗しちゃいました．%q です", err)
		if _, _, err := client.PostMessage(channel, slack.MsgOptionText(msg, false)); err != nil {
			log.Printf("post message failed, msg: %+v, %v", txt, err)
		}
		return
	}
	if _, err = client.UploadFile(
		slack.FileUploadParameters{
			Reader:         resp.image,
			Filetype:       UploadFileType,
			Filename:       UploadImageFileName,
			Title:          resp.title,
			InitialComment: resp.comment,
			Channels:       []string{channel},
		}); err != nil {
		log.Printf("upload lattice image error, %v", err)
	}
}
