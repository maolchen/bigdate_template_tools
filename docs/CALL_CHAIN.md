# 调用链图

本文档描述当前代码中的核心调用链，不是理想架构图。

## 1. 系统总览

```mermaid
flowchart LR
  UI[React Web] --> API[Go HTTP Server]
  API --> AUTH[auth_store.go]
  API --> UCFG[user_config_store.go]
  API --> HANDLERS[handlers.go]
  API --> AI[ai_handlers.go / ai_store.go]
  API --> CHECKER[checker/checker.go]
  HANDLERS --> CFG[config/config.go]
  HANDLERS --> GEN[generator/*]
  AI --> CFG
  AI --> UCFG
  GEN --> TPL[templates/**.tmpl]
  CFG --> YAML[config.yaml / user yaml]
  UCFG --> UFILES[data/users/**]
  AUTH --> AFILES[data/auth + data/users/users.json]
  AI --> AIFILES[data/ai/**]
  GEN --> OUT[output/users/<username>/**]
```

## 2. 登录与初始化链路

```mermaid
sequenceDiagram
  participant Browser
  participant App as App.tsx
  participant AuthAPI as /api/auth/*
  participant AuthStore as auth_store.go
  participant UserStore as user_config_store.go
  participant FS as FileSystem

  Browser->>App: 页面加载
  App->>AuthAPI: GET /api/auth/me
  AuthAPI->>AuthStore: withAuth + principalFromToken
  alt 已登录
    AuthAPI-->>App: user + activeTemplateId
    App->>AuthAPI: GET /api/config
    AuthAPI->>UserStore: loadConfigForPrincipal
    UserStore->>FS: 读取 root config 或 user config
    AuthAPI-->>App: 配置对象
    App->>AuthAPI: GET /api/config/version
    AuthAPI-->>App: 根配置版本
  else 未登录
    AuthAPI-->>App: 401
    App-->>Browser: 显示登录页
  end
```

## 3. 配置读写链路

```mermaid
flowchart TD
  A[页面修改配置] --> B[App.tsx handleConfigChange]
  B --> C[web/src/api/config.ts saveConfig]
  C --> D[PUT /api/config]
  D --> E[server/handlers.go handleConfig]
  E --> F[loadConfigForPrincipal]
  E --> G[config.SaveConfig]
  G --> H[root config.yaml 或 user config.yaml]
  E --> I{当前用户角色}
  I -->|admin| J[syncRootConfigToAllUsers]
  I -->|user| K[syncUserBackupsFromMainConfig]
  D --> L[返回 success]
```

## 4. 主配置增量同步链路

```mermaid
flowchart TD
  A[App.tsx 轮询 /api/config/version] --> B{根配置版本变化?}
  B -->|否| C[无动作]
  B -->|是| D[显示主配置更新提示]
  D --> E[POST /api/config/sync-main]
  E --> F[handleSyncMainConfig]
  F --> G[config.LoadConfig root config.yaml]
  F --> H{admin ?}
  H -->|yes| I[syncRootConfigToAllUsers]
  H -->|no| J[loadUserActiveConfig]
  J --> K[diffConfigChangesFromMain / syncMissingServicesFromMain]
  F --> L[返回 sync 摘要 + 最新 version + config]
```

## 5. 配置模板管理链路

```mermaid
flowchart TD
  A[配置管理页] --> B[GET /api/config-templates]
  B --> C[refreshUserTemplateIndex]
  C --> D[data/users/<username>/configs/templates_index.json]

  E[保存为备份模板] --> F[POST /api/config-templates]
  F --> G[复制 config.yaml -> <id>_config.yaml]
  G --> H[refreshUserTemplateIndex]

  I[切换模板] --> J[POST /api/config-templates/:id/switch]
  J --> K[applyMainServiceSyncToUserTemplate]
  K --> L[swapTemplateFiles]
  L --> M[主模板/备份模板互换]
  M --> N[loadUserActiveConfig]
  N --> O[返回新 config]
```

## 6. 生成输出链路

```mermaid
flowchart TD
  A[点击生成配置] --> B[POST /api/generate]
  B --> C[handleGenerate]
  C --> D[loadConfigForPrincipal]
  D --> E[generator.BuildServiceInstances]
  E --> F[generator.GenerateOutputs]
  F --> G[扫描 templates/<service>/*.tmpl]
  G --> H[generator.RenderTemplate]
  H --> I[写入 output/users/<username>/<nodeIP>/<service>/...]
  I --> J[返回生成统计和结果]
```

## 7. 模板渲染链路

```mermaid
flowchart LR
  A[Config] --> B[BuildServiceInstances]
  B --> C[ServiceInstance[]]
  C --> D[Context]
  D --> E[BuildTemplateFuncMap]
  E --> F[ParseTemplateContent / RenderTemplate]
  F --> G[输出文件]
```

## 8. AI 模板工作台链路

```mermaid
sequenceDiagram
  participant UI as AITemplatePage
  participant API as /api/ai/template/session/:id/message
  participant AIH as ai_handlers.go
  participant AIP as ai_prompt_skills.go
  participant Client as ai_client.go
  participant Store as ai_store.go
  participant FS as data/ai + templates

  UI->>API: POST message?stream=1
  API->>AIH: handleAISessionMessage / Stream
  AIH->>Store: loadAISession
  AIH->>AIP: buildAIChatRequest
  AIP-->>AIH: prompt + skill + examples + attachments
  AIH->>Client: ChatCompletion / StreamChatCompletion
  Client-->>AIH: 模型原始响应
  AIH->>AIH: parseAIModelResponse
  AIH->>AIH: enrichConfigPatchForDrafts
  AIH->>Store: saveAISession
  AIH-->>UI: 会话消息 / 草稿 / 配置补丁 / 问题

  UI->>API: POST save
  API->>AIH: handleAISessionSave
  AIH->>Store: saveDraftFiles
  Store->>FS: 写入 templates/**/*.tmpl
  alt applyConfigPatch=true
    AIH->>AIH: applyConfigPatchToConfig
    AIH->>FS: 保存 config.yaml
  end
  AIH-->>UI: savedFiles + configIssues + nextActions
```

## 9. 模板编辑器链路

```mermaid
flowchart TD
  A[TemplateEditorPage] --> B[GET /api/templates/editor/tree]
  B --> C[buildTemplateEditorTree]
  C --> D[读取 templates 目录树]

  A --> E[GET /api/templates/editor/file?path=...]
  E --> F[读取模板文件]

  A --> G[PUT /api/templates/editor/file]
  G --> H[validateDraftTemplate]
  H --> I[validateDraftTemplateContract]
  I --> J[写入 templates/**/*.tmpl]

  A --> K[POST /api/templates/editor/item]
  K --> L[创建目录或 .tmpl 文件]

  A --> M[DELETE /api/templates/editor/item]
  M --> N[删除文件或目录]
```

## 10. 系统设置与用户管理链路

```mermaid
flowchart TD
  A[Sidebar 系统设置] --> B[修改密码]
  B --> C[POST /api/auth/change-password]
  C --> D[changePassword]
  D --> E[data/users/users.json]

  A --> F[用户管理]
  F --> G[GET /api/users]
  G --> H[listUsers]
  F --> I[POST /api/users]
  I --> J[createUser]
  J --> K[ensureUserConfigForLogin]
  F --> L[PUT /api/users/:username]
  L --> M[updateUser]
```

## 11. 文件落盘链路

```mermaid
flowchart LR
  A[auth_store.go] --> B[data/users/users.json]
  A --> C[data/auth/sessions.json]
  D[user_config_store.go] --> E[data/users/<username>/configs/config.yaml]
  D --> F[data/users/<username>/configs/<id>_config.yaml]
  D --> G[data/users/<username>/configs/templates_index.json]
  H[ai_store.go] --> I[data/ai/sessions/<sessionId>.json]
  H --> J[data/ai/uploads/**]
  K[handleAISessionSave] --> L[templates/**/*.tmpl]
  M[generator/output.go] --> N[output/users/<username>/**]
```
