# LX04 客户端源码

从 axcom/xiaobot 的固定提交派生，完整保留编译所需代码。仓库根目录的 `make test`、`make build` 是维护入口。

| 路径 | 用途 |
| --- | --- |
| `cmd/lx04-agent` | 最终语音入口，本机 logcat、规则分流、音频恢复 |
| `cmd/lx04-bootstrap` | 小米登录、验证码、精确匹配设备；必须传入私密设备选择文件 |
| `cmd/lx04-netcheck` | DeepSeek 模型与可选短对话检查，会产生 API 请求 |
| `cmd/lx04-searchcheck` | 官方搜索与模型合成检查，会产生付费请求 |
| `devicevoice` | 原生识别事件、天气/家居分流、Android 8.1 音频控制 |
| `deepseekweb`、`voiceanswer` | 有次数上限的联网搜索和工具调用 |
| `miservice` | 小米认证、令牌与 TTS 接口，保留独立 MIT 许可证 |
| `runtimeenv` | 音箱进程专属 DNS |

旧的 `cmd/*.go`、云端轮询、JS 引擎和 Web UI 属于继承代码，不是最终部署入口。`lx04-agent` 不启动 Web UI，不向模型开放 JS、shell 或家居执行工具。不要对旧 `cmd` 目录直接执行 `go build ./cmd`，其内有多个独立 main。

上游说明留在 [README.upstream.md](README.upstream.md)，其中其他机型/旧入口的说明不能代替[本项目部署指南](../docs/部署指南.md)。变更和来源见 [源码来源与许可](../THIRD_PARTY.md)。
