module xiaobot

go 1.24.0

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/GopeedLab/gopeed v1.8.2
	github.com/dop251/goja v0.0.0-20251008123653-cf18d89f3cf6
	github.com/dop251/goja_nodejs v0.0.0-20251015164255-5e94316bedaf
	github.com/google/gopacket v1.1.19
	github.com/imroc/req/v3 v3.55.0
	github.com/texttheater/golang-levenshtein/levenshtein v0.0.0-20200805054039-cae8b0eaed6c
	go.uber.org/zap v1.27.0
)

require (
	github.com/roylee0704/gron v0.0.0-20160621042432-e78485adab46 // indirect
	go.uber.org/mock v0.5.2 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
)

require (
	github.com/abema/go-mp4 v1.4.1
	github.com/andybalholm/brotli v1.2.0
	github.com/cloudflare/circl v1.6.1
	github.com/dlclark/regexp2 v1.11.4
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible
	github.com/google/go-querystring v1.1.0
	github.com/google/pprof v0.0.0-20250423184734-337e5dd93bb4
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/hajimehoshi/go-mp3 v0.3.4
	github.com/icholy/digest v1.1.0
	github.com/icza/bitio v1.1.0
	github.com/klauspost/compress v1.18.0
	github.com/mewkiz/flac v1.0.13
	github.com/mewkiz/pkg v0.0.0-20250417130911-3f050ff8c56d
	github.com/mewpkg/term v0.0.0-20241026122259-37a80af23985
	github.com/quic-go/qpack v0.5.1
	github.com/quic-go/quic-go v0.53.0
	github.com/refraction-networking/utls v1.7.3
	github.com/youpy/go-riff v0.1.0
	github.com/youpy/go-wav v0.3.2
	github.com/zaf/g711 v0.0.0-20190814101024-76a4a538f52b
	golang.org/x/crypto v0.39.0
	golang.org/x/net v0.41.0
	golang.org/x/sys v0.33.0
	golang.org/x/text v0.26.0
	ninego/filelog v0.0.0-00010101000000-000000000000
	ninego/log v0.0.0-00010101000000-000000000000
)

replace ninego/log => ../third_party/ninego/log

replace ninego/filelog => ../third_party/ninego/filelog
