package slack

// Message represents a message.
type Message struct {
	ID      uint64 `json:"id"`
	Type    string `json:"type"`
	SubType string `json:"subtype"`
	Channel string `json:"channel"`
	UserID  string `json:"user"`
	Text    string `json:"text"`
	Time    int64  `json:"time"`
}

// TextBody returns the body of the message.
func (m Message) TextBody() string {
	matches := reMsg.FindStringSubmatch(m.Text)
	if len(matches) == 3 {
		return matches[2]
	}
	return ""
}

// MentionID returns the mention id of this message.
func (m Message) MentionID() string {
	matches := reMsg.FindStringSubmatch(m.Text)
	if len(matches) == 3 {
		return matches[1]
	}
	return ""
}
