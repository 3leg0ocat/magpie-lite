# magpie-lite

[yetone/magpie](https://github.com/yetone/magpie) 的 Windows x64 发行版，基于上游发布版本，修改如下。

## 修改

- **修复**：Windows 10 下 Claude Code 交互模式截断长消息的问题（[yetone/magpie#5205](https://github.com/yetone/magpie/issues/5205)）。
- **更新**：移除自动更新、更新检查和 What's new。
- **隐私**：不发送使用统计。
- **供应商**：“添加供应商”仅列出以下选项，并移除赞助商推荐。
  - 官方：Anthropic、OpenAI、Google Gemini、DeepSeek、xAI、Kimi、智谱 / Z.ai、MiniMax、阿里云百炼
  - 本地：Ollama、LM Studio、oMLX、MLX-Serve
  - Remote magpie
- **资源库**：仅保留 MCP 服务器和技能；Instructions、Hooks、子代理和 RTK 不再显示，也不再同步到各 agent。

订阅、插件、自定义供应商和已有配置不受影响。

## 第三方组件

Microsoft.Windows.Console.ConPTY 1.25.260930003，MIT 许可，见 [ConPTY-LICENSE.txt](https://github.com/3leg0ocat/magpie-lite/blob/lite/windows/build/lite/ConPTY-LICENSE.txt)。
