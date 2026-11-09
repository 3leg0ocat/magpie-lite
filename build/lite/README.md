# magpie-lite（Windows x64）

基于 [yetone/magpie](https://github.com/yetone/magpie) 的 Windows 精简版。对原项目的修改：

- 修复 Windows 10 上 Claude Code 交互模式丢失长消息的问题（yetone/magpie#5205）：使用随程序附带的微软新版 ConPTY，不再使用系统自带的 ConPTY。
- 不检查、不下载、不安装 magpie 自身的更新。
- 不发送每日用户统计，不弹出 What's new。
- 添加供应商时只列出 Anthropic、OpenAI、Google Gemini、DeepSeek、xAI、Kimi、智谱/Z.ai、MiniMax、阿里云百炼，本地的 Ollama、LM Studio、oMLX、MLX-Serve，以及 Remote magpie；不显示赞助商。
- 资源库只保留 MCP 服务器和技能。

订阅、插件、自定义供应商和已保存的配置不受影响。
