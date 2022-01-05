## listmonk-messenger

Lightweight HTTP server to handle webhooks from [listmonk](https://listmonk.app) and forward it to different messengers.

### Supported messengers

* Pinpoint

### Development

* Build binary
```
make build
```

* Change config.toml and tweak messenger config

Run the binary which starts a server on :8082
```
./listmonk-messenger.bin --config config.toml --msgr pinpoint
```

* Setting up webhooks
![](/screenshots/listmonk-setting-up-webhook.png)

* Add messenger specific subscriber atrributes in listmonk
![](/screenshots/listmonk-add-subsriber-attrib.png)

* Add plain text template
![](/screenshots/listmonk-plain-text-template.png)

* Change campaign messenger
![](/screenshots/listmonk-change-campaign-mgr.png)