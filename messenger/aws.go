package messenger

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

// awsCfg holds the AWS credentials and region shared by the SES and Pinpoint
// messengers.
type awsCfg struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Region    string `json:"region"`
	// Endpoint overrides the AWS service endpoint, e.g. for a VPC endpoint or a
	// LocalStack mock. Empty uses the real AWS endpoints.
	Endpoint string `json:"endpoint"`

	// RoleARN, when set, assumes this IAM role via STS and uses the resulting
	// temporary credentials. The base credentials for AssumeRole are the static
	// keys above if set, else the default credential chain.
	RoleARN string `json:"role_arn"`
	// ExternalID is passed to AssumeRole for third-party access. See
	// https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html
	ExternalID string `json:"external_id"`
	// RoleSessionName defaults to "listmonk-messenger" when empty.
	RoleSessionName string `json:"role_session_name"`
}

// newAWSSession builds an AWS session, assuming RoleARN via STS when it is set.
func newAWSSession(c awsCfg) (*session.Session, error) {
	config := &aws.Config{
		MaxRetries: aws.Int(3),
	}
	if c.AccessKey != "" && c.SecretKey != "" {
		config.Credentials = credentials.NewStaticCredentials(c.AccessKey, c.SecretKey, "")
	}
	if c.Region != "" {
		config.Region = &c.Region
	}
	if c.Endpoint != "" {
		config.Endpoint = &c.Endpoint
		// Path-style addressing keeps single-host mocks reachable.
		config.S3ForcePathStyle = aws.Bool(true)
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	if c.RoleARN != "" {
		sessionName := c.RoleSessionName
		if sessionName == "" {
			sessionName = "listmonk-messenger"
		}

		creds := stscreds.NewCredentials(sess, c.RoleARN, func(p *stscreds.AssumeRoleProvider) {
			p.RoleSessionName = sessionName
			if c.ExternalID != "" {
				p.ExternalID = aws.String(c.ExternalID)
			}
		})
		sess, err = session.NewSession(config.Copy().WithCredentials(creds))
		if err != nil {
			return nil, err
		}
	}

	return sess, nil
}

// checkCredentials verifies the session's credentials resolve via STS.
func checkCredentials(sess *session.Session) error {
	_, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	return err
}
