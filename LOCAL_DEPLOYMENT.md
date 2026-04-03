# 本地部署指南

## 项目结构

```
project/
├── main.go              # 源代码（统一入口）
├── server-bin           # Linux/Mac 可执行文件
├── server.exe           # Windows 可执行文件
├── config.yaml          # 配置文件
├── templates/           # 模板文件目录
└── web/
    └── dist/           # 前端构建产物（必需）
        ├── index.html
        └── assets/
```

## 快速开始

### 方式一：使用预编译二进制文件

**Linux/Mac:**

```bash
# 1. 下载文件
wget https://example.com/server-bin
chmod +x server-bin

# 2. 确保目录结构
project/
├── server-bin
├── config.yaml
├── templates/
└── web/
    └── dist/

# 3. 运行
./server-bin --web

# 访问 http://localhost:5000/
```

**Windows:**

```powershell
# 1. 下载文件
# 保存为 server.exe

# 2. 确保目录结构
project\
├── server.exe
├── config.yaml
├── templates\
└── web\
    └── dist\

# 3. 运行
.\server.exe --web

# 访问 http://localhost:5000/
```

### 方式二：从源码编译

#### 1. 安装 Go

访问 [https://golang.org/dl/](https://golang.org/dl/) 下载并安装 Go 1.21+

验证安装：
```bash
go version
```

#### 2. 编译

**Linux/Mac:**
```bash
go build -o server-bin main.go
```

**Windows:**
```powershell
go build -o server.exe main.go
```

#### 3. 构建前端（可选）

如果需要自己构建前端：

```bash
# 安装依赖
pnpm install

# 构建
pnpm run build
```

## 运行模式

### 命令行模式

直接生成配置文件，不启动 Web 服务：

```bash
./server-bin
```

输出：
- 配置文件生成到 `output/` 目录
- 按节点 IP 组织输出结构

### Web 模式

启动 Web 服务，提供可视化配置界面：

```bash
# 默认端口 5000
./server-bin --web

# 指定端口
./server-bin --web --port=8080
```

访问：
- Web 界面：http://localhost:5000/
- API 文档：http://localhost:5000/api/config

## 目录说明

| 目录/文件 | 说明 | 是否必需 |
|-----------|------|----------|
| `main.go` | Go 源代码 | 开发环境 |
| `server-bin` / `server.exe` | 编译后的可执行文件 | ✅ 必需 |
| `config.yaml` | 配置文件 | ✅ 必需 |
| `templates/` | 模板文件目录 | ✅ 必需 |
| `web/dist/` | 前端构建产物 | Web 模式必需 |
| `output/` | 输出目录 | 自动创建 |

## 常见问题

### 1. 访问 http://localhost:5000/ 显示 404

**原因：** 前端文件缺失或路径错误

**解决方案：**

1. 检查启动日志中的前端目录路径
2. 确保 `web/dist/index.html` 存在
3. 确保 `server-bin` 和 `web/` 在同一目录

```bash
# 检查文件
ls -la web/dist/
```

### 2. 前端目录路径问题

程序支持两种目录结构：

**结构 1（推荐）：**
```
project/
├── server-bin
├── config.yaml
├── templates/
└── web/
    └── dist/
```

**结构 2：**
```
project/
├── server/
│   └── server-bin
├── config.yaml
├── templates/
└── web/
    └── dist/
```

程序会自动检测并使用正确的路径。

### 3. API 正常但前端 404

**现象：**
- `http://localhost:5000/api/config` 正常返回 JSON
- `http://localhost:5000/` 返回 404

**原因：** 静态文件服务路由配置问题

**解决方案：**

1. 确认 `web/dist/index.html` 存在
2. 检查文件权限
3. 查看启动日志中的"前端静态文件目录"输出

### 4. 端口被占用

```bash
# 查找占用端口的进程
netstat -ano | findstr :5000  # Windows
lsof -i :5000                  # Linux/Mac

# 结束进程（替换 PID）
taskkill /PID [进程ID] /F      # Windows
kill -9 [PID]                  # Linux/Mac

# 或使用其他端口
./server-bin --web --port=8080
```

### 5. Windows 防火墙提示

首次运行 Windows 版本时，可能会弹出防火墙提示。

**解决方法：**
- 选择"允许访问"
- 或在防火墙设置中添加例外规则

## 完整构建流程

```bash
# 1. 克隆项目
git clone <repository-url>
cd project

# 2. 构建前端
pnpm install
pnpm run build

# 3. 编译后端
go build -o server-bin main.go

# 4. 运行
./server-bin --web
```

## 开发环境

### 热重载开发

```bash
# 前端开发（端口 5000）
pnpm dev

# 后端开发
go run main.go --web
```

### 代码检查

```bash
# Go 代码检查
go vet ./...
go fmt ./...

# 前端代码检查
pnpm lint
```

## 部署建议

### 生产环境

1. 使用 systemd 管理（Linux）
2. 配置反向代理（Nginx）
3. 启用 HTTPS
4. 配置日志轮转

### systemd 示例

```ini
[Unit]
Description=BigData Config Generator
After=network.target

[Service]
Type=simple
User=app
WorkingDirectory=/opt/bigdata-config
ExecStart=/opt/bigdata-config/server-bin --web --port=5000
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Nginx 反向代理

```nginx
server {
    listen 80;
    server_name config.example.com;

    location / {
        proxy_pass http://localhost:5000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```
