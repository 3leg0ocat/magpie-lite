# magpie-lite（Windows x64）

基于上游 [yetone/magpie](https://github.com/yetone/magpie) 的精简分支。只在上游发布标签之上加几个小提交，便于跟随上游。

发布物是单个 `magpie-lite-<版本>-windows-x64.exe`，可以改名为 `magpie.exe` 使用。

## 和上游的区别

| 提交 | 内容 |
|---|---|
| `claudetui: …ConPTY…` | Claude Code 交互模式使用微软新版 ConPTY（`conpty.dll` + `OpenConsole.exe`）。Windows 10 自带的 ConPTY 会吞掉括号粘贴标记，导致长消息被截断（yetone/magpie#5205）。本构建把这两个文件和它们的 MIT 许可嵌在 exe 里，首次运行 Claude Code 时写到 exe 旁的 `conpty\<版本>\`，只使用这一份。写入或加载失败时直接报错，不会静默退回系统 ConPTY。不带 `liteconpty` 构建标签时，若 exe 旁有 `conpty\` 目录则使用它，否则与上游一致。 |
| `update: a build with selfUpdate off …` | 构建参数 `-X …/internal/update.selfUpdate=off`：不检查、不下载、不安装 magpie 自身的更新，也不要求 Remote magpie 更新；设置页只显示版本号。插件和 Library 中 skill 的更新不受影响。 |
| `provider: a build with trimmed on …` | 构建参数 `-X …/internal/provider.trimmed=on`：添加供应商时只列出 Anthropic、OpenAI、Google Gemini、DeepSeek、xAI、Kimi、智谱/Z.ai、MiniMax、阿里云百炼，本地的 Ollama、LM Studio、oMLX、MLX-Serve，以及 Remote magpie；不拉取也不显示赞助商列表。已保存的其他供应商照常使用和编辑；订阅、插件和自定义供应商不受影响。 |
| `library: …tab` | Library 页只保留 MCP 服务器和技能，不再显示 Instructions、Hooks、子代理和 RTK。已有内容、已经分发给各 agent 的文件和已装的 RTK hook 保持原样。Agents 页为子代理选模型的设置不受影响。 |
| `lite: Windows package …` | 本目录：打包脚本、ConPTY 许可、本说明，以及 `.github/workflows/lite-release.yml`。 |

不带上述构建标签和参数时，第 2、3 项不生效，行为与上游相同；上游测试因此无需改动。

ConPTY 组件：NuGet 包 `Microsoft.Windows.Console.ConPTY` 1.25.260930003（x64），微软签名，MIT 许可（`ConPTY-LICENSE.txt`）。打包脚本校验包的 SHA-256，该值与 NuGet 目录登记的 SHA-512 对应同一文件。

## 构建

在 Windows 的 Git Bash 中运行（需要 Go、Python 和 curl）：

```sh
build/lite/package-windows.sh
```

产物为 `dist/magpie-lite-<版本>-windows-x64.exe` 及其 `.sha256`。脚本下载固定版本的 ConPTY 包并校验哈希，也可以用 `CONPTY_NUPKG=<已下载的 .nupkg>` 指定本地文件。组件在构建期间放在 `internal/claudetui/conpty-bundle/`，构建结束后删除，不进仓库。

## 升级

把新 exe 放到正在运行的 `magpie.exe` 的位置：先把旧文件改名（例如 `magpie.exe.bak`），再把新文件复制为 `magpie.exe`。正在运行的 magpie 发现文件被替换后，会等在途请求结束，再自动重启成新版本，不截断进行中的 Claude 回合。Windows 上重启的几秒内，新请求可能连不上。

## 跟随上游

```sh
git fetch upstream --tags
git rebase --onto <新的上游标签> <当前的上游标签> lite/windows
# 解决冲突后
go vet ./internal/claudetui ./internal/update ./internal/provider
go test ./internal/update ./internal/provider -run '^(TestOff|TestTrimmed)'
git tag <新的上游标签>-lite.1
build/lite/package-windows.sh
```

验收至少包括：在 Windows 10 上用真实 Claude Code 交互模式发送一段 8 KB 以上的消息，对照 Claude Code transcript 中记录的用户消息，确认内容完整。

发布标签形如 `v0.1.3357-lite.1`。推送到自己的仓库后，`lite-release.yml` 会构建并创建 GitHub Release（fork 默认不运行 Actions，需要时在仓库设置中启用）。启用 Actions 后，上游的 `release.yml` 也会因 `v*` 标签运行并失败（没有上游的密钥），`test.yml` 等也会运行；可在 Actions 页面单独停用它们。

## 已知限制

- 之前从赞助商列表添加的供应商仍可正常请求，但编辑时显示为自定义供应商。
- exe 所在目录需要可写，用于放置 `conpty\<版本>\`。
