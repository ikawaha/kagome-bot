package client

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Bot struct {
	client *socketmode.Client
	id     string
}

func New(appToken, botToken string, debug bool) (*Bot, error) {
	LoadDict()
	c := slack.New(
		botToken,
		slack.OptionAppLevelToken(appToken),
		slack.OptionDebug(debug),
	)
	resp, err := c.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("slack-client-token is invalid: %v\n", err)
	}
	bot := socketmode.New(c, socketmode.OptionDebug(debug))
	return &Bot{
		client: bot,
		id:     fmt.Sprintf("<@%v>", resp.UserID),
	}, nil
}

func (bot Bot) Run(ctx context.Context) error {
	h := socketmode.NewSocketmodeHandler(bot.client)
	h.HandleEvents(slackevents.Message, newMessageTokenizeHandlerFunc(ctx, bot.id))
	h.Handle(socketmode.EventTypeSlashCommand, newSlashCommandTokenizeHandlerFunc(ctx))
	h.HandleDefault(defaultHandler)

	return h.RunEventLoopContext(ctx)
}
