package client

import (
	"context"
)

const (
	develVersion   = "devel"
	unknownVersion = "unknown"
)

type botVersionCtxKey struct{}

func WithBotVersion(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, botVersionCtxKey{}, v)
}

func BotVersionFromContext(ctx context.Context) string {
	v, ok := ctx.Value(botVersionCtxKey{}).(string)
	if !ok {
		return develVersion
	}
	return v
}

type tokenizerVersionCtxKey struct{}

func WithTokenizerVersion(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, tokenizerVersionCtxKey{}, v)
}

func TokenizerVersionFromContext(ctx context.Context) string {
	v, ok := ctx.Value(tokenizerVersionCtxKey{}).(string)
	if !ok {
		return unknownVersion
	}
	return v
}
