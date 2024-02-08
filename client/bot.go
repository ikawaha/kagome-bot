package client

import (
	"context"
	"fmt"
	debugpkg "runtime/debug"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type Bot struct {
	client   *socketmode.Client
	id       string
	versions Versions
}

type Versions struct {
	Kagome string
	Bot    string
}

func New(appToken, botToken string, debug bool) (*Bot, error) {
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

type botInfo struct {
	botVersion       string
	tokenizerVersion string
}

func (bot Bot) Run(ctx context.Context) error {
	v := bot.version()
	info := botInfo{
		botVersion:       v.bot,
		tokenizerVersion: v.tokenizer,
	}
	h := socketmode.NewSocketmodeHandler(bot.client)
	h.HandleEvents(slackevents.Message, newMessageTokenizeHandlerFunc(ctx, bot.id, info))
	h.Handle(socketmode.EventTypeSlashCommand, newSlashCommandTokenizeHandlerFunc(ctx, info))
	h.HandleDefault(defaultHandler)

	return h.RunEventLoopContext(ctx)
}

type version struct {
	bot       string
	tokenizer string
}

func (bot Bot) version() version {
	ret := version{
		bot:       "devel",
		tokenizer: "unknown",
	}
	info, ok := debugpkg.ReadBuildInfo()
	if ok {
		ret.bot = info.Main.Version
	}
	for _, dep := range info.Deps {
		if strings.Contains(dep.Path, "ikawaha/kagome/v2") {
			ret.tokenizer = dep.Version
		}
	}
	return ret
}
