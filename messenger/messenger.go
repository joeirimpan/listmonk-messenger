package messenger

import (
	"net/textproto"

	"github.com/knadh/listmonk/models"
)

type Messenger interface {
	Name() string
	Push(Message) error
	Flush() error
	Close() error
}

// Message is the message pushed to a Messenger.
type Message struct {
	From        string
	To          []string
	Subject     string
	ContentType string
	Body        []byte
	AltBody     []byte
	Headers     textproto.MIMEHeader
	Attachments []Attachment

	Subscriber models.Subscriber

	// Campaign is generally the same instance for a large number of subscribers.
	Campaign *models.Campaign
}

// Attachment represents a file or blob attachment that can be
// sent along with a message by a Messenger.
type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte
}
