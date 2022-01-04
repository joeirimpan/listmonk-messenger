package messenger

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/pinpoint"
)

var (
	channelType = "SMS"
)

type pinpointCfg struct {
	AppID       string `json:"app_id"`
	AccessKey   string `json:"access_key"`
	SecretKey   string `json:"secret_key"`
	Region      string `json:"region"`
	MessageType string `json:"message_type"`
	SenderID    string `json:"sender_id"`
	Log         bool   `json:"log"`
}

type pinpointMessenger struct {
	cfg    pinpointCfg
	client *pinpoint.Pinpoint
}

func (p pinpointMessenger) Name() string {
	return "pinpoint"
}

// Push sends the sms through pinpoint API.
func (p pinpointMessenger) Push(msg Message) error {
	phone, ok := msg.Subscriber.Attribs["phone"].(string)
	if !ok {
		return fmt.Errorf("could not find subscriber phone")
	}

	body := string(msg.Body)
	payload := &pinpoint.SendMessagesInput{
		ApplicationId: &p.cfg.AppID,
		MessageRequest: &pinpoint.MessageRequest{
			Addresses: map[string]*pinpoint.AddressConfiguration{
				phone: {
					ChannelType: &channelType,
				},
			},
			MessageConfiguration: &pinpoint.DirectMessageConfiguration{
				SMSMessage: &pinpoint.SMSMessage{
					Body:        &body,
					MessageType: &p.cfg.MessageType,
					SenderId:    &p.cfg.SenderID,
				},
			},
		},
	}

	out, err := p.client.SendMessages(payload)
	if err != nil {
		return err
	}

	if p.cfg.Log {
		for phone, result := range out.MessageResponse.Result {
			log.Printf("successfully sent sms to %s: %#+v", phone, result)
		}
	}

	return nil
}

func (p pinpointMessenger) Flush() error {
	return nil
}

func (p pinpointMessenger) Close() error {
	return nil
}

// NewPinpoint creates new instance of pinpoint
func NewPinpoint(cfg []byte) (Messenger, error) {
	var c pinpointCfg
	if err := json.Unmarshal(cfg, &c); err != nil {
		return nil, err
	}

	if c.AppID == "" {
		return nil, fmt.Errorf("invalid app_id")
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
	svc := pinpoint.New(sess,
		aws.NewConfig().
			WithCredentials(credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, "")).
			WithRegion(c.Region),
	)

	return pinpointMessenger{
		client: svc,
		cfg:    c,
	}, nil
}
