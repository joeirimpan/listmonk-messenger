package messenger

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/knadh/smtppool"
)

const (
	ContentTypeHTML  = "html"
	ContentTypePlain = "plain"
)

type sesCfg struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	Log       bool   `json:"log"`
}

type sesMessenger struct {
	cfg    sesCfg
	client *ses.SES
}

func (s sesMessenger) Name() string {
	return "ses"
}

// Push sends the sms through pinpoint API.
func (s sesMessenger) Push(msg Message) error {
	// convert attachments to smtppool.Attachments
	a := make([]smtppool.Attachment, 0, len(msg.Attachments))
	for i := 0; i < len(msg.Attachments); i++ {
		a[i] = smtppool.Attachment{
			Filename: msg.Attachments[i].Name,
			Header:   msg.Attachments[i].Header,
			Content:  msg.Attachments[i].Content,
		}
	}

	email := smtppool.Email{
		From:        msg.From,
		To:          msg.To,
		Subject:     msg.Subject,
		Sender:      msg.From,
		Headers:     msg.Headers,
		Attachments: a,
	}

	switch {
	case msg.ContentType == ContentTypePlain:
		email.Text = msg.Body
	case msg.ContentType == ContentTypeHTML:
		email.HTML = msg.Body
	}

	emailB, err := email.Bytes()
	if err != nil {
		return err
	}

	to := make([]*string, 0, len(msg.To))
	for i := 0; i < len(msg.To); i++ {
		to = append(to, &msg.To[i])
	}

	input := &ses.SendRawEmailInput{
		Source:       &msg.From,
		Destinations: to,
		RawMessage: &ses.RawMessage{
			Data: emailB,
		},
	}

	out, err := s.client.SendRawEmail(input)
	if err != nil {
		return err
	}

	if s.cfg.Log {
		log.Printf("successfully sent email to %s: %#+v", msg.Subscriber.Email, out)
	}

	return nil
}

func (s sesMessenger) Flush() error {
	return nil
}

func (s sesMessenger) Close() error {
	return nil
}

// NewAWSSES creates new instance of pinpoint
func NewAWSSES(cfg []byte) (Messenger, error) {
	var c sesCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, err
	}

	if c.Region == "" {
		return nil, fmt.Errorf("invalid region")
	}
	if c.AccessKey == "" {
		return nil, fmt.Errorf("invalid access_key")
	}
	if c.SecretKey == "" {
		return nil, fmt.Errorf("invalid secret_key")
	}

	sess := session.Must(session.NewSession())
	svc := ses.New(sess,
		aws.NewConfig().
			WithCredentials(credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, "")).
			WithRegion(c.Region),
	)

	return sesMessenger{
		client: svc,
		cfg:    c,
	}, nil
}
