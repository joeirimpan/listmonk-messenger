package messenger

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/francoispqt/onelog"
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

	logger *onelog.Logger
}

func (s sesMessenger) Name() string {
	return "ses"
}

// Push sends the sms through pinpoint API.
func (s sesMessenger) Push(msg Message) error {
	// convert attachments to smtppool.Attachments
	var files []smtppool.Attachment
	if msg.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(msg.Attachments))
		for i := 0; i < len(msg.Attachments); i++ {
			files[i] = smtppool.Attachment{
				Filename: msg.Attachments[i].Name,
				Header:   msg.Attachments[i].Header,
				Content:  make([]byte, len(msg.Attachments[i].Content)),
			}
			copy(files[i].Content, msg.Attachments[i].Content)
		}
	}

	email := smtppool.Email{
		From:        msg.Campaign.FromEmail,
		Subject:     msg.Subject,
		Sender:      msg.From,
		Headers:     msg.Headers,
		Attachments: files,
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

	input := &ses.SendRawEmailInput{
		Source:       &email.From,
		Destinations: []*string{&msg.Subscriber.Email},
		RawMessage: &ses.RawMessage{
			Data: emailB,
		},
	}

	out, err := s.client.SendRawEmail(input)
	if err != nil {
		return err
	}

	if s.cfg.Log {
		s.logger.InfoWith("successfully sent email").String("email", msg.Subscriber.Email).String("results", fmt.Sprintf("%#+v", out)).Write()
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
func NewAWSSES(cfg []byte, l *onelog.Logger) (Messenger, error) {
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
		logger: l,
	}, nil
}
