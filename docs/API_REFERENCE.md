# 接口文档

本文档基于当前代码实际实现整理，不是规划接口。

- 后端入口：`server/server.go`
- 认证方式：Cookie Session（`config_generator_token`）
- 普通响应：JSON
- 下载与附件接口除外

## 1. 认证与用户管理

### POST /api/auth/login

登录并写入 Cookie。

请求：

```json
{
  "username": "admin",
  "password": "Admin@123"
}
```

响应：

```json
{
  "success": true,
  "user": {
    "username": "admin",
    "role": "admin",
    "mustChangePassword": true
  },
  "sync": {
    "addedGlobalKeys": [],
    "globalAdded": 0,
    "addedServices": [],
    "serviceTopAdded": 0,
    "serverConfigAdded": 0
  }
}
```

### POST /api/auth/logout

退出登录并清理 Cookie。

### GET /api/auth/me

获取当前登录用户信息。

响应：

```json
{
  "user": {
    "username": "admin",
    "role": "admin",
    "mustChangePassword": false
  },
  "activeTemplateId": "config"
}
```

### POST /api/auth/change-password

修改当前用户密码。

请求：

```json
{
  "oldPassword": "Admin@123",
  "newPassword": "NewPassword123"
}
```

### GET /api/users

仅管理员可用。获取用户列表。

### POST /api/users

仅管理员可用。创建用户。

请求：

```json
{
  "username": "alice",
  "password": "Password123",
  "role": "user",
  "enabled": true
}
```

### PUT /api/users/:username

仅管理员可用。更新用户状态、角色、重置密码。

请求：

```json
{
  "role": "user",
  "enabled": true,
  "resetPassword": "NewPassword123",
  "mustChangePassword": true
}
```

## 2. 配置管理

### GET /api/config

获取当前用户可编辑配置。

- 管理员：根目录 `config.yaml`
- 普通用户：`data/users/<username>/configs/config.yaml`

### PUT /api/config

保存当前用户可编辑配置。

管理员保存后会同步增量更新到所有普通用户空间。

### POST /api/config/reload

从当前用户配置文件重新加载配置。

### GET /api/config/version

获取根目录 `config.yaml` 的版本信息，用于普通用户轮询检测主配置变化。

响应：

```json
{
  "path": ".../config.yaml",
  "mtime": "2026-04-15T08:00:00Z",
  "mtimeUnixNano": 1713168000000000000,
  "size": 12345,
  "sha256": "..."
}
```

### POST /api/config/sync-main

将根配置中的新增项同步到当前用户配置。

- 管理员调用：同步到所有普通用户空间
- 普通用户调用：同步到自己当前主模板

响应字段：

- `sync.addedGlobalKeys`
- `sync.removedGlobalKeys`
- `sync.addedServiceTopServices`
- `sync.addedServerConfigServices`
- `config`
- `version`

### POST /api/config/sync-main-delete

危险操作。按根配置清理当前普通用户配置中的删除项。

- 管理员不可调用

## 3. 配置模板管理

### GET /api/config-templates

获取当前用户的配置模板列表。

### POST /api/config-templates

从当前主模板创建备份模板。

请求：

```json
{
  "id": "project_4node",
  "remark": "4 节点项目模板"
}
```

### POST /api/config-templates/:id/switch

切换主模板。

当前实现是“主模板与目标备份模板互换”。

响应：

```json
{
  "success": true,
  "activeTemplateId": "config",
  "sync": {
    "addedGlobalKeys": [],
    "globalAdded": 0,
    "addedServices": [],
    "serviceTopAdded": 0,
    "serverConfigAdded": 0
  },
  "config": {}
}
```

### DELETE /api/config-templates/:id

删除备份模板。

- `config` 主模板不可删除

## 4. 描述管理与引用检查

### GET /api/descriptions/global

获取全局配置字段描述。

### POST /api/descriptions/global

保存全局配置字段描述。

### GET /api/descriptions/:serviceName

获取服务配置描述。

### POST /api/descriptions/:serviceName

保存服务配置描述。

### POST /api/check-references

检查变量/节点/服务是否被模板引用。

请求：

```json
{
  "type": "vars",
  "key": "http_port",
  "service": "elasticsearch"
}
```

响应：

```json
{
  "hasReferences": true,
  "references": [
    {
      "path": "templates/elasticsearch/install.sh.tmpl",
      "service": "elasticsearch",
      "details": ["..."]
    }
  ]
}
```

## 5. 生成与输出

### POST /api/generate

根据当前用户配置生成输出。

### GET /api/output

获取当前用户输出文件列表。

### GET /api/output/file?path=...

读取当前用户某个输出文件内容。

### GET /api/output/download

下载当前用户输出目录打包结果。

## 6. 模板浏览与模板编辑器

### GET /api/templates

获取模板文件列表。

### GET /api/templates/editor/tree

获取 `templates/` 目录树。

响应：

```json
{
  "root": "templates",
  "readonly": false,
  "nodes": [
    {
      "name": "elasticsearch",
      "path": "elasticsearch",
      "type": "dir",
      "children": []
    }
  ]
}
```

### GET /api/templates/editor/file?path=...

读取某个模板文件。

### PUT /api/templates/editor/file

保存模板文件。

- 仅管理员允许
- 仅允许 `.tmpl`
- 会做模板语法校验与模板占位符约束校验

请求：

```json
{
  "path": "elasticsearch/install.sh.tmpl",
  "content": "..."
}
```

### POST /api/templates/editor/item

创建模板文件或目录。

- 仅管理员允许

请求：

```json
{
  "path": "spark3/install.sh.tmpl",
  "type": "file",
  "content": "{{/* new template */}}"
}
```

或：

```json
{
  "path": "spark3",
  "type": "dir"
}
```

### DELETE /api/templates/editor/item?path=...

删除模板文件或目录。

- 仅管理员允许
- 删除目录时递归删除

## 7. AI 设置与模型

### GET /api/ai/settings

获取 AI 设置。

### PUT /api/ai/settings

保存 AI 设置。

请求：

```json
{
  "baseUrl": "https://dashscope.aliyuncs.com/compatible-mode/v1",
  "model": "qwen3.5-plus",
  "apiKey": "...",
  "clearApiKey": false
}
```

### POST /api/ai/settings/test

测试 AI 设置连通性。

### GET /api/ai/models

拉取模型列表。

### GET /api/ai/rules

读取 AI 全局规则。

### PUT /api/ai/rules

保存 AI 全局规则。

## 8. AI Skill 与提示目录

### GET /api/ai/catalog

获取 AI skill 与 examples 索引。

### GET /api/ai/skill-file?path=...

读取一个 skill 文件内容。

### PUT /api/ai/skill-file

保存 skill 文件内容。

### GET /api/ai/skills

获取自定义 skill 列表。

### POST /api/ai/skills

创建自定义 skill。

### PUT /api/ai/skills/:id

更新自定义 skill。

### DELETE /api/ai/skills/:id

删除自定义 skill。

## 9. AI 模板工作台

### GET /api/ai/template/session

获取 AI 会话摘要列表。

### POST /api/ai/template/session

创建新会话。

### DELETE /api/ai/template/session?id=...

删除会话。

也支持：

- `DELETE /api/ai/template/session/:id`

### GET /api/ai/template/session/:id

获取完整会话。

### PUT /api/ai/template/session/:id/meta

更新会话标题。

### POST /api/ai/template/session/:id/preview

预览本次发送前的 prompt trace、skill、example、附件上下文。

### POST /api/ai/template/session/:id/message

发送对话消息。

- 支持普通 JSON 响应
- 支持 `?stream=1` 流式返回

请求：

```json
{
  "message": "请生成 elasticsearch 安装模板",
  "sessionRules": "",
  "selectedDraftPaths": [],
  "selectedSkillIds": ["core/output_json"],
  "model": "qwen3.5-plus"
}
```

### POST /api/ai/template/session/:id/upload

上传附件。

- 支持文本与图片
- 上传后作为下一次提问的 pending context

### DELETE /api/ai/template/session/:id/drafts

删除草稿。

请求：

```json
{
  "paths": ["templates/elasticsearch/install.sh.tmpl"],
  "removeFromDisk": true
}
```

### POST /api/ai/template/session/:id/save

保存审核通过的草稿文件到 `templates/`。

- 仅管理员允许
- 可选同步配置补丁
- 若配置补丁存在阻塞问题，模板文件仍可能已保存，但配置不会同步

请求：

```json
{
  "files": [
    {
      "path": "templates/elasticsearch/install.sh.tmpl",
      "content": "...",
      "reason": "新增安装脚本",
      "needsReview": false
    }
  ],
  "applyConfigPatch": true
}
```

响应示例：

```json
{
  "savedFiles": ["templates/elasticsearch/install.sh.tmpl"],
  "createdDirs": ["templates/elasticsearch"],
  "configApplied": false,
  "appliedServices": [],
  "configIssues": [],
  "nextActions": []
}
```

### GET /api/ai/template/session/:id/attachment/:attachmentId

下载或读取附件。

### DELETE /api/ai/template/session/:id/attachment/:attachmentId

删除附件。

## 10. 认证与权限说明

### 10.1 认证方式

后端通过以下顺序解析登录态：

1. `Authorization: Bearer <token>`
2. Cookie：`config_generator_token`

前端当前统一使用 Cookie，会话通过 `credentials: include` 发送。

### 10.2 权限规则

- `admin`
  - 可用所有接口
- `user`
  - 不能访问 `/api/users*`
  - 不能保存模板到正式目录
  - 不能修改模板目录树
  - 模板编辑器仅只读

## 11. 主要落盘文件

- 用户：`data/users/users.json`
- 会话：`data/auth/sessions.json`
- 用户配置：`data/users/<username>/configs/*.yaml`
- 用户模板索引：`data/users/<username>/configs/templates_index.json`
- AI 设置：`data/ai/settings.json`
- AI 规则：`data/ai/template_rules.md`
- AI 会话：`data/ai/sessions/<sessionId>.json`
- AI 上传：`data/ai/uploads/...`
- 正式模板：`templates/**/*.tmpl`
