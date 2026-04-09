# AI 模板工作台（v1）

本文档说明 `AI 模板` 页面新增能力与使用方式。

## 1. 核心能力

- 对话生成/改写 `templates/<service>/*.tmpl` 草稿
- 上传或粘贴附件参与本轮上下文
- 流式回复与思考状态展示
- 草稿人工编辑后保存到 `templates/`
- 可选保存时同步 `serviceTop` / `serverConfig.vars` 建议
- 历史会话管理（搜索、改名、删除、切换）
- Skill 管理（编辑 `data/ai/skills/*/*.md`）

## 2. 新增交互（本次更新）

- 复杂粘贴：在输入框 `Ctrl+V` 可直接粘贴图片/文件（浏览器可读到文件对象时）
- 拖拽上传：可把文本文件或图片直接拖到输入区
- 模型切换：输入区上方新增模型输入框，支持自由输入和建议列表
  - 优先级：本次请求 `model` > 会话已选模型 > AI 设置默认模型
  - 会话会记住本次实际使用模型，刷新/切换会话后可继续使用

## 3. 附件类型与限制

- 文本附件：`.sh` `.tmpl` `.conf` `.yaml` `.yml` `.xml` `.properties` `.json` `.md` `.txt`
- 图片附件：`.png` `.jpg` `.jpeg` `.webp`
- 单文件限制：文本 1MB，图片 5MB
- 单次消息附件上限：最多 3 个，其中图片最多 2 张

## 4. API Key 与加密

- 页面填写的是**真实 API Key 明文**（例如供应商给你的 `sk-...`）
- 后端不会回传明文，只回传掩码
- 落盘文件：`data/ai/settings.json`
  - 仅保存 `apiKeyEncrypted`
- 加密方式：`CONFIG_GENERATOR_AI_MASTER_KEY` 经过 SHA-256 派生 32 字节密钥后，使用 AES-256-GCM 加密

## 5. 使用示例（Elasticsearch）

1. 打开 `AI 模板` 页面，先在 `AI 设置` 填 `Base URL / Model / API Key`
2. 在输入框提问：
   - `帮我生成一个 elasticsearch 3 节点集群安装脚本模板，目标 templates/elasticsearch/install.sh.tmpl`
3. 如需参考材料，拖拽上传脚本/配置，或粘贴截图
4. AI 输出草稿后可直接点 `编辑` 进入右侧 Monaco 修改
5. 点击 `保存当前` 或 `保存全部`
6. 若返回了配置补丁建议，可点击 `保存并同步配置`

## 6. 仓库清理说明

- `.gocache/` 属于本地 Go 编译缓存，已加入 `.gitignore`
- 不应再提交到仓库
