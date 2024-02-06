package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome-dict-ipa-neologd"
	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome-dict/uni"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

const (
	GraphvizCmd = "dot"
	CmdTimeout  = 25 * time.Second

	UploadFileType      = "png"
	UploadImageFileName = "lattice.png"
)

type dictKind string

const (
	ipaDict     dictKind = "ipa"
	uniDict     dictKind = "uni"
	neologdDict dictKind = "neologd"
)

type tokenizeResponse struct {
	title   string
	image   io.Reader
	comment string
}

func init() {
	_ = ipa.Dict()
	_ = uni.Dict()
	_ = ipaneologd.Dict()
}

func newDict(d dictKind) *dict.Dict {
	switch d {
	case ipaDict:
		return ipa.Dict()
	case uniDict:
		return uni.Dict()
	case neologdDict:
		return ipaneologd.Dict()
	default:
		return ipa.Dict()
	}
}

func tokenize(ctx context.Context, txt string, dict dictKind) (*tokenizeResponse, error) {
	t, err := tokenizer.New(newDict(dict))
	if err != nil {
		return nil, fmt.Errorf("tokenizer initialization failed, %w", err)
	}
	if _, err := exec.LookPath(GraphvizCmd); err != nil {
		return nil, fmt.Errorf("command %v is not installed in your $PATH", GraphvizCmd)
	}
	var buf bytes.Buffer
	ctx, cancel := context.WithTimeout(ctx, CmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, GraphvizCmd, "-T"+UploadFileType)
	r0, w0 := io.Pipe()
	cmd.Stdin = r0
	cmd.Stdout = &buf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("process done with error, %w", err)
	}
	tokens := t.AnalyzeGraph(w0, txt, tokenizer.Normal)
	if err := w0.Close(); err != nil {
		return nil, fmt.Errorf("pipe close error, %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("process done with error, %w", err)
	}
	return &tokenizeResponse{
		title:   txt,
		image:   &buf,
		comment: "```" + yield(tokens) + "```",
	}, nil
}

func yield(tokens []tokenizer.Token) string {
	var buf bytes.Buffer
	for _, token := range tokens {
		if token.Class == tokenizer.DUMMY {
			continue
		}
		_, _ = fmt.Fprintf(&buf, "%s\t%s\n", token.Surface, strings.Join(token.Features(), ","))
	}
	return buf.String()
}
