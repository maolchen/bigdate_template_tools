# Windows 本地部署文件清单

## 下载以下文件和目录

### 必需文件

- `server.exe` - Windows 可执行文件
- `config.yaml` - 配置文件
- `main.go` - 命令行版本（可选，用于直接生成配置）

### 必需目录

- `templates/` - 模板文件目录（所有模板）
- `web/` - 前端源码目录
  - `dist/` - 前端构建产物（必需，包含 index.html 和 assets/）

### 文件结构

将下载的文件按以下结构组织：

```
your-project/
├── server.exe          # 主程序
├── config.yaml         # 配置文件
├── templates/          # 模板目录
│   ├── dnsmasq/
│   ├── hadoop3/
│   ├── kafka/
│   └── ... (其他模板)
└── web/
    └── dist/           # 前端构建产物（必需）
        ├── index.html
        └── assets/
```

## 运行步骤

1. 打开命令提示符或 PowerShell
2. 进入项目目录：
   ```powershell
   cd your-project
   ```
3. 运行服务：
   ```powershell
   server.exe
   ```
4. 访问：
   - Web 界面：http://localhost:5000/
   - API：http://localhost:5000/api/config

## 常见问题

### 404 错误

如果访问 http://localhost:5000/ 显示 404，检查：

1. `web/dist/index.html` 文件是否存在
2. 查看启动日志中的"前端静态文件目录"路径
3. 确认 `server.exe` 和 `web/` 在同一目录

### Windows 防火墙

首次运行可能需要允许防火墙访问：
- Windows 会弹出提示，选择"允许访问"

### 端口占用

如果 5000 端口被占用：
```powershell
# 查找占用端口的进程
netstat -ano | findstr :5000

# 结束进程（替换 PID）
taskkill /PID [进程ID] /F
```

## 从命令行直接生成配置（不启动 Web）

```powershell
# 使用命令行版本
server.exe generate
```

这会读取 `config.yaml` 并在 `output/` 目录生成配置文件。
