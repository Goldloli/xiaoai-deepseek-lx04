# XiaoSeek LX04

Add DeepSeek Q&A and web search to a Xiaomi XiaoAI touchscreen speaker while retaining native weather, weather-based clothing advice and Mi Home commands.

[中文文档](README.md) · [Source and licenses](THIRD_PARTY.md) · [Technical report, Chinese](docs/改造过程与修改技术报告.md)

The client runs **on the speaker**. No always-on Mac, NAS or Raspberry Pi is required. DeepSeek inference and search use cloud APIs; wake-word recognition, ASR and TTS retain Xiaomi's native service chain. This is not an offline LLM.

## Status

Validated on **one Chinese LX04, stock firmware 2.42.118, Android 8.1, MT8167, ARMv7, approximately 1 GB RAM**. The owner confirmed knowledge answers, native weather/clothing advice and search worked. Other models and firmware versions are unverified. Public-source device selection and DNS were generalized after the original installation; the existing speaker was not reflashed during publication.

Local final-ASR log events feed conservative native-first routing. Open questions go to DeepSeek. Temporary playback master mute suppresses competing native speech; saved state is restored by speech events, bounded fallbacks and recovery paths. The native NLP engine still runs independently: this is a sidecar, not a protocol interception proxy.

Fresh questions can pre-search; other questions allow a read-only `web_search` tool. Each question has a maximum of two searches and three model requests. The model has no shell or home-control tool. Search and model calls incur provider charges.

## Build

Install Git, Make, Python 3 and the Go version in `.go-version`:

```sh
git clone https://github.com/Goldloli/xiaoai-deepseek-lx04.git
cd xiaoai-deepseek-lx04
make test
make build
```

Outputs in `dist/`: four ARMv7 commands and `SHA256SUMS`. Tests and builds do not connect to a device. `client/` includes the complete modified client; the two pinned local Go modules are in `third_party/ninego/`.

Deployment requires authenticated root ADB and a compatible audio/logging implementation. Read the [deployment guide](docs/部署指南.md) and [compatibility/recovery notes](docs/兼容性与恢复.md) before device work. Historical boot scripts contain device-specific layouts and verified stock hashes. They are engineering records, **not universal flashers**. Backups, ROM images, credentials, tokens, keys and private logs are not distributed.

## Scope and licenses

Boot candidate generation, V1–V6 ADB work, bounded flashing/recovery scripts, startup templates, account verification, routing, audio handling and search are open. Default: Apache-2.0. The mtkclient patch and five hardware scripts retain GPL-3.0-only; MiService retains MIT. See [NOTICE](NOTICE) and [THIRD_PARTY.md](THIRD_PARTY.md).

Limitations: one-device validation; rules can misroute; playback mute affects other audio; no multi-turn memory, custom screen rendering or complete process supervisor. Hardware modification can affect boot and encrypted data. The report distinguishes measured outcomes from untested paths.

Contributions: [CONTRIBUTING.md](CONTRIBUTING.md). Private vulnerability reports: [SECURITY.md](SECURITY.md). Independent community project; not affiliated with Xiaomi or DeepSeek.
