# 使用说明

## 命令行模式（默认）

直接运行二进制文件，生成配置到 `output/` 目录：

```bash
# Linux/Mac
./server-bin

# Windows
server.exe
```

## Web 模式

启动 Web 服务，提供可视化配置界面和 REST API：

```bash
# Linux/Mac（默认端口 5000）
./server-bin --web

# 指定端口
./server-bin --web --port=8080

# Windows
server.exe --web
server.exe --web --port=8080
```

## 访问 Web 界面

启动 Web 模式后，访问：

- **Web 界面**：http://localhost:5000/
- **API 配置**：http://localhost:5000/api/config
- **API 模板**：http://localhost:5000/api/templates

## 本地部署（Windows）

### 文件结构

下载以下文件：

```
your-project/
├── server.exe          # 可执行文件
├── config.yaml         # 配置文件
├── templates/          # 模板目录
└── web/
    └── dist/           # 前端构建产物
        ├── index.html
        └── assets/
```

### 运行

```powershell
# 命令行模式（生成配置）
server.exe

# Web 模式（启动服务）
server.exe --web
```

## API 接口

### GET /api/config
获取当前配置

### PUT /api/config
保存配置

### POST /api/generate
生成配置文件

### GET /api/output
获取生成的文件列表

### GET /api/output/download
下载配置包（ZIP）

### GET /api/templates
获取模板列表

### POST /api/config/reload
从文件重新加载配置

## 环境变量

- `DEPLOY_RUN_PORT`：指定服务端口（优先级低于 --port 参数）
