<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

## listmonk-messenger

Lightweight HTTP server to handle webhooks from [listmonk](https://listmonk.app) and forward it to different messengers.

### Supported messengers

- Pinpoint
- Twilio
- AWS SES - Use `listmonk >= v2.2.0`


### Development

- Build binary

```
make build
```

- Change config.toml and tweak messenger config

Run the binary which starts a server on :8082

```
./listmonk-messenger.bin --config config.toml --msgr pinpoint --msgr ses
```

### AWS credentials (SES & Pinpoint)

The `ses` and `pinpoint` messengers can authenticate to AWS in two ways:

- **Static credentials**: set `access_key` and `secret_key`. If both are left
  empty, the [default AWS credential chain](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html)
  is used (environment variables, EC2/ECS instance profile, etc).

- **AssumeRole (temporary credentials)**: set `role_arn` to have the messenger
  assume an IAM role via STS and use the resulting temporary credentials. The
  base credentials used to call `AssumeRole` are the static keys above if
  provided, otherwise the default credential chain. This is the recommended
  approach over long-lived access keys, and is required for
  [third-party cross-account access](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_common-scenarios_third-party.html).

  | Field               | Required | Description                                                                 |
  | ------------------- | -------- | --------------------------------------------------------------------------- |
  | `role_arn`          | yes      | ARN of the IAM role to assume, e.g. `arn:aws:iam::123456789012:role/ses`.   |
  | `external_id`       | no       | `ExternalId` for third-party access (guards against the confused deputy).   |
  | `role_session_name` | no       | Session name for the assumed role. Defaults to `listmonk-messenger`.        |

### Running tests

```
make test              # unit tests
make test-integration  # spins up a LocalStack mock AWS via docker and runs the integration tests
```

`make test-integration` requires Docker. It brings up LocalStack (defined in
`docker-compose.test.yml`), runs the `integration`-tagged tests against it, and
tears it down afterwards. To run the tests against an already-running mock, set
`LOCALSTACK_ENDPOINT` and invoke `go test -tags=integration ./...` directly.

### Health check

`GET /health` returns `200 OK` and can be used as a liveness/readiness probe for
monitoring.

- Setting up webhooks
  ![](/screenshots/listmonk-setting-up-webhook.png)

- Add messenger specific subscriber atrributes in listmonk
  ![](/screenshots/listmonk-add-subsriber-attrib.png)

- Add plain text template
  ![](/screenshots/listmonk-plain-text-template.png)

- Change campaign messenger
  ![](/screenshots/listmonk-change-campaign-mgr.png)
