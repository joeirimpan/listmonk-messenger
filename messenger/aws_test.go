//go:build integration

// Integration tests for the AWS helpers, run against a LocalStack mock via
// `make test-integration`. The endpoint defaults to http://localhost:4566 and
// can be overridden with LOCALSTACK_ENDPOINT.
package messenger

import (
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
)

const (
	testRegion    = "us-east-1"
	testAccessKey = "test"
	testSecretKey = "test"
)

func localstackEndpoint() string {
	if e := os.Getenv("LOCALSTACK_ENDPOINT"); e != "" {
		return e
	}
	return "http://localhost:4566"
}

func baseCfg() awsCfg {
	return awsCfg{
		AccessKey: testAccessKey,
		SecretKey: testSecretKey,
		Region:    testRegion,
		Endpoint:  localstackEndpoint(),
	}
}

// callerARN resolves the session's identity via STS.
func callerARN(t *testing.T, sess *session.Session) string {
	t.Helper()
	out, err := sts.New(sess).GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		t.Fatalf("GetCallerIdentity: %v", err)
	}
	return aws.StringValue(out.Arn)
}

func TestNewAWSSession_StaticCredentials(t *testing.T) {
	sess, err := newAWSSession(baseCfg())
	if err != nil {
		t.Fatalf("newAWSSession: %v", err)
	}
	if err := checkCredentials(sess); err != nil {
		t.Fatalf("checkCredentials: %v", err)
	}
}

// TestNewAWSSession_AssumeRole checks that role_arn makes the session resolve
// to the assumed-role identity, not the base user.
func TestNewAWSSession_AssumeRole(t *testing.T) {
	const (
		roleName    = "listmonk-test-role"
		sessionName = "listmonk-test-session"
		externalID  = "test-external-id"
	)

	base, err := newAWSSession(baseCfg())
	if err != nil {
		t.Fatalf("base session: %v", err)
	}
	iamSvc := iam.New(base)

	assumePolicy := `{
		"Version": "2012-10-17",
		"Statement": [{"Effect": "Allow", "Principal": {"AWS": "*"}, "Action": "sts:AssumeRole"}]
	}`
	roleOut, err := iamSvc.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(assumePolicy),
	})
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists") {
		t.Fatalf("CreateRole: %v", err)
	}

	var roleARN string
	if roleOut != nil && roleOut.Role != nil {
		roleARN = aws.StringValue(roleOut.Role.Arn)
	} else {
		got, gErr := iamSvc.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
		if gErr != nil {
			t.Fatalf("GetRole: %v", gErr)
		}
		roleARN = aws.StringValue(got.Role.Arn)
	}

	cfg := baseCfg()
	cfg.RoleARN = roleARN
	cfg.RoleSessionName = sessionName
	cfg.ExternalID = externalID

	sess, err := newAWSSession(cfg)
	if err != nil {
		t.Fatalf("newAWSSession (assume role): %v", err)
	}

	// Identity looks like arn:aws:sts::<acct>:assumed-role/<role>/<session>.
	arn := callerARN(t, sess)
	if !strings.Contains(arn, "assumed-role/"+roleName) {
		t.Fatalf("expected assumed-role identity for %q, got %q", roleName, arn)
	}
	if !strings.Contains(arn, sessionName) {
		t.Fatalf("expected session name %q in identity, got %q", sessionName, arn)
	}
}
