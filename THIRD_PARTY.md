# 源码来源与许可证

本项目默认采用 [Apache License 2.0](LICENSE)。以下目录与文件保留各自许可；根许可证不替换上游许可或赋予原厂固件再分发权。

| 内容 | 来源/固定版本 | 许可与处理 |
| --- | --- | --- |
| `client/` 继承代码 | [axcom/xiaobot](https://github.com/axcom/xiaobot/tree/7c98861a48437fcb74ea6483ff365b1e63d7ab1c) | Apache 2.0；保留 `client/LICENSE` 与上游说明 |
| `client/miservice/` | xiaobot 内的 MiService 派生实现 | MIT；保留 [原版权声明](client/miservice/LICENSE)，新增适配亦在此目录按 MIT 提供 |
| `third_party/ninego/log`、`filelog` | [axcom/ninego](https://github.com/axcom/ninego/tree/a4c59c1576bd1eb6c5d80684ad25abd103cf6331) | Apache 2.0；仅包含构建所需两个模块及原许可证 |
| `docs/mtkclient-macos.patch` | [bkerler/mtkclient](https://github.com/bkerler/mtkclient/tree/7cb40e727727dfe3faa065eefb29b16ed29bbdf1) 的修改 | GPL-3.0-only；完整许可见 [GPLv3](LICENSES/GPL-3.0-only.txt) |
| 五个启动/硬件 Python 脚本 | 本项目新增，调用 mtkclient 或与其工作流程组合 | GPL-3.0-only，具体文件如下 |
| `client/go.mod`、`go.sum` 的其他依赖 | Go 模块记录的固定版本 | 构建时下载；各模块保留其自身许可，不将其统一改为 Apache |
| Magisk、ADB、libusb | 外部安装工具 | 不随仓库分发；按各自官方项目许可获取 |
| 原厂镜像、密钥、设备备份 | 用户自己的设备 | 不随源码或 Release 分发 |

GPL-3.0-only 脚本：`scripts/da_to_fastboot.py`、`scripts/prepare_adb_boot.py`、`scripts/prepare_seccfg.py`、`scripts/lx04_boot_trial.py`、`scripts/revise_adb_boot.py`。

2026-10-24 的客户端修改包括本机 ASR 入口、规则分流、音频静音/恢复、DNS、DeepSeek 搜索及工具循环、小米验证/令牌隔离、TLS 校验、测试。公开整理额外引入私密设备选择文件和相对依赖路径。原有上游版权头保持不变；可应用的变更补丁见 [xiaobot-local.patch](docs/xiaobot-local.patch)，机器可读来源见 [UPSTREAM.json](third_party/UPSTREAM.json)。

相关背景项目：xiaobot 基于 [longbai/xiaobot](https://github.com/longbai/xiaobot) 与 [xiaogpt](https://github.com/yihong0618/xiaogpt) 的工作；MiService 由上游项目转换及适配。本项目不声称这些基础代码是原创。

Magisk 参考 [官方 v30.7 源码](https://github.com/topjohnwu/Magisk/tree/v30.7)，Platform-Tools 从 [Google 官方页面](https://developer.android.com/tools/releases/platform-tools) 获取。发布内容只有源码与文档；暂不提供预编译程序，避免将原机二进制误作通用刷机包。
