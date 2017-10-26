package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/ikawaha/kagome/tokenizer"
)

const (
	graphvizCmd = "circo"
	cmdTimeout  = 25 * time.Second
)

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
		return nil, nil, errors.New("graphviz timeout")
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
