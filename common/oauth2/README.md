# common/oauth2

A lightweight, dependency-free OAuth2 client built on the Go standard library
(`net/http` + `encoding/json`).

## Features

- Authorization-code flow: generate auth URLs, exchange codes for tokens,
  refresh tokens.
- Normalized user info (`UserInfo`) across multiple providers.
- Preconfigured providers: Google, GitHub, WeChat, DingTalk, Feishu.
- CSRF `state` generation and validation.
- No third-party dependencies (test-only `testify`).

## Quick start

```go
provider := oauth2.GoogleProvider("client-id", "client-secret", "https://example.com/callback")
client := oauth2.NewClient(provider)

state := oauth2.GenerateState()
authURL := client.AuthURL(state)
// redirect the user to authURL ...

tok, err := client.Exchange(ctx, code)
if err != nil {
    log.Fatal(err)
}

user, err := client.GetUser(ctx, tok)
if err != nil {
    log.Fatal(err)
}
fmt.Println(user.ID, user.Name, user.Email)
```

## Providers

| Provider  | AuthURL                                                        | TokenURL                                                                  | UserInfoURL                                                          |
| --------- | -------------------------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Google    | `https://accounts.google.com/o/oauth2/v2/auth`                 | `https://oauth2.googleapis.com/token`                                     | `https://www.googleapis.com/oauth2/v2/userinfo`                      |
| GitHub    | `https://github.com/login/oauth/authorize`                     | `https://github.com/login/oauth/access_token`                             | `https://api.github.com/user`                                        |
| WeChat    | `https://open.weixin.qq.com/connect/qrconnect`                 | `https://api.weixin.qq.com/sns/oauth2/access_token`                       | `https://api.weixin.qq.com/sns/userinfo`                             |
| DingTalk  | `https://login.dingtalk.com/oauth2/auth`                       | `https://api.dingtalk.com/v1.0/oauth2/userAccessToken`                    | `https://api.dingtalk.com/v1.0/contact/users/me`                     |
| Feishu    | `https://open.feishu.cn/open-apis/authen/v1/index`             | `https://open.feishu.cn/open-apis/authen/v1/access_token`                 | `https://open.feishu.cn/open-apis/authen/v1/user_info`               |

## License

MIT
