package messenger

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sts"
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
		for _, f := range msg.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
	}

	email := smtppool.Email{
		From:        msg.Campaign.FromEmail,
		To:          []string{msg.Subscriber.Email},
		Subject:     msg.Subject,
		Sender:      msg.From,
		Headers:     msg.Headers,
		Attachments: files,
	}

	switch {
	case msg.ContentType == ContentTypePlain:
		email.Text = msg.Body
	default:
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

func checkCredentials(sess *session.Session) bool {
	// Create a SES service client.
	svc := sts.New(sess)
	// Call the GetCallerIdentity API to check credentials
	params := &sts.GetCallerIdentityInput{}
	_, err := svc.GetCallerIdentity(params)
	return err != nil
}

// NewAWSSES creates new instance of pinpoint
func NewAWSSES(cfg []byte, l *onelog.Logger) (Messenger, error) {
	var c sesCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, err
	}

	config := &aws.Config{
		MaxRetries: aws.Int(3),
	}
	if c.AccessKey != "" && c.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, "")
	}
	if c.Region != "" {
		config.Region = &c.Region
	}

	var sess = session.Must(session.NewSession(config))
	if !checkCredentials(sess) {
		return nil, fmt.Errorf("invalid credentials")
	}

	svc := ses.New(sess)
	return sesMessenger{
		client: svc,
		cfg:    c,
		logger: l,
	}, nil
}
