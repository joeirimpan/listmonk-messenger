package messenger

import (
	"encoding/json"
	"fmt"
	
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/francoispqt/onelog"
)

type twilioCfg struct {
	AccountID   string `json:"account_id"`
	AuthToken   string `json:"auth_token"`
	SenderID    string `json:"sender_id"`
	UploadPath  string `json:"upload_path"`
	Log         bool   `json:"log"`
}

type twilioMessenger struct {
	cfg    twilioCfg
	client *twilio.RestClient

	logger *onelog.Logger
}

func (t twilioMessenger) Name() string {
	return "twilio"
}

// Push sends the sms through twilio API.
func (t twilioMessenger) Push(msg Message) error {
	phone, ok := msg.Subscriber.Attribs["phone"].(string)
	if !ok {
		return fmt.Errorf("could not find subscriber phone")
	}

	body := string(msg.Body)
	payload := &twilioApi.CreateMessageParams{}
	payload.SetTo(phone)
	payload.SetFrom(t.cfg.SenderID)
	payload.SetBody(body)
	if msg.Attachments != nil {
		media := make([]string, 0, len(msg.Attachments))
		for _, f := range msg.Attachments {
			media = append(media,fmt.Sprintf("%s/%s",t.cfg.UploadPath,f.Name))
		}
		if (len(media) > 0) {
			payload.SetMediaUrl(media)
		}
	}

	out, err := t.client.Api.CreateMessage(payload)
	if err != nil {
		return err
	}

	if t.cfg.Log {
		response, _ := json.Marshal(*out)
		t.logger.InfoWith("successfully sent sms").String("phone", phone).String("result", string(response)).Write()
	}

	return nil
}

func (t twilioMessenger) Flush() error {
	return nil
}

func (t twilioMessenger) Close() error {
	return nil
}

// NewTwilio creates new instance of twilio
func NewTwilio(cfg []byte, l *onelog.Logger) (Messenger, error) {
	var c twilioCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, err
	}

	if c.AccountID == "" {
		return nil, fmt.Errorf("invalid account_id")
	}
	if c.AuthToken == "" {
		return nil, fmt.Errorf("invalid auth_token")
	}
	if c.SenderID == "" {
		return nil, fmt.Errorf("invalid sender_id")
	}
	if c.UploadPath == "" {
		return nil, fmt.Errorf("invalid upload_path")
	}

	svc := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: c.AccountID,
		Password: c.AuthToken,
	})

	return twilioMessenger{
		client: svc,
		cfg:    c,
		logger: l,
	}, nil
}
