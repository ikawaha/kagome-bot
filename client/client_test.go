package client

import (
	"testing"
)

func TestClientUserNameEmpty(t *testing.T) {
	b := Client{Users: map[string]string{}}
	if n := b.UserName(""); n != "" {
		t.Errorf("got %v, expected empty", n)
	}
}

func TestClientUserName(t *testing.T) {
	m := map[string]string{
		"U03CP354N": "foo",
		"U02J1PU37": "baa",
	}
	b := Client{Users: m}
	for id, user := range m {
		if u := b.UserName(id); u != user {
			t.Errorf("got %v, expected empty", u, user)
		}
	}
}
