# constants

Shared constants for the ling-base platform: environment names, database drivers, server defaults, timezones, bootstrap parameters, and ANSI codes.

## Categories

- Environment names: `ENV_LOCAL`, `ENV_DEV`, `ENV_PROD`
- Database drivers: `DBDriverSQLite`, `DBDriverMySQL`, `DBDriverPG`
- Server defaults: `DefaultServerAddr`, `DefaultUploadDir`, timeout constants
- Timezone IANA names: `TimezoneShanghai`, `TimezoneUTC`, `TimezoneNewYork`, ...
- Session defaults: `DefaultSessionExpireDays`, `DefaultSessionRandomKeyLen`
- SIP defaults: `DefaultSIPHost`, `DefaultSIPPort`, `DefaultSIPLocalIP`
- Bootstrap: banner generation, SQL parsing, JWKS key management
- ANSI terminal codes: `ANSIReset`, banner gradient colors

## Key functions

- `IsProdMode(mode)` -- reports whether a MODE value requests production-strict behaviour

## Usage

```go
import "github.com/LingByte/ling-base/common/constants"

if constants.IsProdMode(cfg.Server.Mode) {
    // enable production-strict checks
}
tz := constants.TimezoneShanghai // "Asia/Shanghai"
```

## License

MIT
