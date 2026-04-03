# 本地部署指南

## 项目结构说明

### 两个 main.go 的区别

1. **根目录 `/main.go`** - 命令行工具
   - 用途：直接运行生成配置，不启动 Web 服务
   - 运行方式：`go run main.go` 或编译后直接执行
   - 输出：生成 `output/` 目录

2. **`server/main.go`** - Web API 服务
   - 用途：提供 REST API 和 Web 界面
   - 运行方式：`cd server && go run main.go`
   - 提供：前端界面 + API 接口

### 目录结构

```
project/
├── server/
│   ├── main.go          # Web 服务的 Go 代码
│   ├── web/
│   │   └── dist/        # 前端构建产物（必须）
│   ├── config/          # 配置文件（可选，默认使用根目录的 config.yaml）
│   └── templates/       # 模板文件（可选，默认使用根目录的 templates/）
├── web/                 # 前端源码
├── templates/           # 模板文件
├── config.yaml          # 配置文件
└── server-bin           # 预编译的二进制文件（Linux）
```

---

## Windows 本地部署步骤

### 方式一：使用预编译二进制文件

#### 1. 准备文件

从项目下载以下文件和目录：

**必需文件：**
- `server-bin` (Linux 二进制，不适用于 Windows)
- 或者在 Windows 上编译：

#### 2. 在 Windows 上编译

需要先安装 Go 环境：
1. 下载 Go: https://golang.org/dl/
2. 安装后验证：`go version`

编译：
```powershell
# 进入 server 目录
cd server

# 编译为 Windows 二进制
go build -o server.exe main.go
```

#### 3. 准备必要目录

确保以下目录存在（相对于 `server.exe`）：

**方案 A：将文件放在 server.exe 同级目录**
```
项目根目录/
├── server.exe          # 可执行文件
├── config.yaml         # 配置文件
├── templates/          # 模板目录
└── web/
    └── dist/           # 前端构建产物
        ├── index.html
        └── assets/
```

**方案 B：将文件放在 server.exe 的父目录**
```
项目根目录/
├── server/
│   └── server.exe      # 可执行文件
├── config.yaml         # 配置文件
├── templates/          # 模板目录
└── web/
    └── dist/           # 前端构建产物
```

#### 4. 运行服务

```powershell
# Windows CMD
server.exe

# 或 PowerShell
.\server.exe
```

服务启动后会输出：
```
============================================
大数据平台配置生成器 API 服务
============================================
工作目录: [工作目录路径]
配置文件: [配置文件路径]
模板目录: [模板目录路径]
输出目录: [输出目录路径]
端口: 5000
API: http://localhost:5000/api/config
Web: http://localhost:5000/
============================================
```

#### 5. 访问服务

- Web 界面：http://localhost:5000/
- API 文档：http://localhost:5000/api/config
- 模板列表：http://localhost:5000/api/templates

---

## 常见问题排查

### 问题 1：访问 http://localhost:5000/ 显示 404

**原因：**
- 前端目录路径不正确
- `web/dist` 目录不存在或缺少文件

**排查步骤：**

1. 查看启动日志中的"前端静态文件目录"输出：
   ```
   检查前端目录: [路径]
   ✓ 前端静态文件目录: [路径]
     文件数: X
   ```

2. 如果看到 `✗ 前端静态文件目录不存在`，说明路径配置错误

3. 确保 `web/dist` 目录包含 `index.html` 文件：
   ```powershell
   dir web\dist\index.html
   ```

4. 如果缺少 `index.html`，需要从项目复制前端构建产物

**解决方案：**

将 `server/web/dist/` 目录复制到项目根目录下的 `web/dist/`：
```powershell
# 创建目录
mkdir web
mkdir web\dist

# 复制文件（假设你从项目下载了 server/web/dist）
xcopy /E /I server\web\dist web\dist
```

---

### 问题 2：Windows 路径问题

**现象：**
- 启动日志中路径显示为 `C:\path\to\project\server\web\dist`
- 但实际文件在 `C:\path\to\project\web\dist`

**解决方案：**

修改工作目录结构，确保符合以下之一：

**结构 1（推荐）：**
```
项目根目录/
├── server.exe
├── config.yaml
├── templates/
└── web/
    └── dist/
```

**结构 2：**
```
项目根目录/
├── config.yaml
├── templates/
├── web/
│   └── dist/
└── server/
    └── server.exe
```

程序会自动检测这两种结构。

---

### 问题 3：API 正常但前端 404

**现象：**
- `http://localhost:5000/api/config` 正常返回 JSON
- `http://localhost:5000/` 返回 404

**原因：**
静态文件服务路由配置问题。

**解决方案：**

1. 确认 `web/dist/index.html` 存在
2. 检查文件权限（Windows 上应该没有这个问题）
3. 使用绝对路径启动服务：
   ```powershell
   cd C:\path\to\project
   server\server.exe
   ```

---

## 开发环境构建

### 构建前端

```bash
# 安装依赖
pnpm install

# 构建
pnpm run build

# 产物在 web/dist/
```

### 编译 Go 后端

**Linux:**
```bash
cd server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../server-bin main.go
```

**Windows:**
```powershell
cd server
go build -o server.exe main.go
```

**macOS:**
```bash
cd server
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o server-darwin main.go
```

---

## 清理项目

运行清理脚本删除测试文件：
```bash
bash clean.sh
```

或手动删除：
- `output/` 目录
- `tmp/` 目录
- `config-generator` 二进制文件
- `config-simple.yaml`
- `run.log`
- 根目录的 `index.html`（如果有）
