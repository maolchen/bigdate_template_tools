# Windows 本地部署指南

## 下载文件

从项目下载以下文件和目录：

### 必需文件

- `server.exe` - Windows 可执行文件（约 11MB）
- `config.yaml` - 配置文件

### 必需目录

- `templates/` - 模板文件目录（所有模板）
- `web/` - 前端源码目录
  - `dist/` - 前端构建产物（必需，包含 index.html 和 assets/）

## 目录结构

将下载的文件按以下结构组织：

```
your-project\
├── server.exe          # 主程序
├── config.yaml         # 配置文件
├── templates\          # 模板目录
│   ├── dnsmasq\
│   ├── hadoop3\
│   ├── kafka\
│   └── ... (其他模板)
└── web\
    └── dist\           # 前端构建产物（必需）
        ├── index.html
        └── assets\
```

## 运行步骤

### 1. 打开命令提示符

按 `Win + R`，输入 `cmd`，回车

### 2. 进入项目目录

```cmd
cd C:\path\to\your-project
```

### 3. 运行服务

```cmd
server.exe --web
```

### 4. 访问服务

打开浏览器访问：
- Web 界面：http://localhost:5000/
- API：http://localhost:5000/api/config

## 运行模式

### 命令行模式

直接生成配置文件：

```cmd
server.exe
```

这会：
- 读取 `config.yaml`
- 生成配置到 `output/` 目录
- 生成完成后自动退出

### Web 模式

启动 Web 服务：

```cmd
# 默认端口 5000
server.exe --web

# 指定端口
server.exe --web --port=8080
```

## 常见问题

### 1. 404 错误

**现象：** 访问 http://localhost:5000/ 显示 404

**原因：** 前端文件缺失或路径错误

**排查步骤：**

1. 检查启动日志中的"前端静态文件目录"输出
2. 确保 `web\dist\index.html` 文件存在
3. 确保 `server.exe` 和 `web\` 在同一目录

```cmd
dir web\dist\index.html
```

**解决方案：**

如果缺少 `index.html`，需要从项目复制前端构建产物：

```cmd
# 创建目录
mkdir web
mkdir web\dist

# 复制文件（从下载的包中）
xcopy /E /I [下载包路径]\web\dist web\dist
```

### 2. Windows 防火墙

**现象：** 首次运行时弹出防火墙提示

**解决方法：**
- 点击"允许访问"
- 或在防火墙设置中手动添加例外

### 3. 端口占用

**现象：** 启动失败，提示端口被占用

**排查步骤：**

```cmd
# 查找占用端口的进程
netstat -ano | findstr :5000

# 结束进程（替换 PID）
taskkill /PID [进程ID] /F

# 或使用其他端口
server.exe --web --port=8080
```

### 4. 找不到配置文件

**现象：** 启动时提示"警告: 无法加载配置文件"

**原因：** `config.yaml` 不在正确位置

**解决方案：**

确保 `config.yaml` 和 `server.exe` 在同一目录：

```cmd
dir config.yaml
```

### 5. 找不到模板文件

**现象：** 生成配置时提示"没有对应的模板目录"

**原因：** `templates/` 目录缺失或不完整

**解决方案：**

确保 `templates/` 目录和所有子目录都存在：

```cmd
dir templates
dir templates\hadoop3
```

### 6. 路径问题

**现象：** 启动日志显示路径为 `C:\path\to\project\web\dist` 但实际文件在 `C:\path\to\project\web\dist`

**原因：** 路径配置问题

**解决方案：**

确保目录结构符合要求：

```
your-project\
├── server.exe
├── config.yaml
├── templates\
└── web\
    └── dist\
```

程序会自动检测这种结构。

## 从源码编译

### 1. 安装 Go

1. 访问 [https://golang.org/dl/](https://golang.org/dl/)
2. 下载 Windows 安装包（go1.21.x.windows-amd64.msi）
3. 双击安装
4. 验证安装：
   ```cmd
   go version
   ```

### 2. 安装 Node.js（可选，如需构建前端）

1. 访问 [https://nodejs.org/](https://nodejs.org/)
2. 下载并安装 LTS 版本
3. 安装 pnpm：
   ```cmd
   npm install -g pnpm
   ```

### 3. 编译

```cmd
# 进入项目目录
cd C:\path\to\project

# 编译 Windows 二进制
go build -o server.exe main.go

# 运行
server.exe --web
```

### 4. 构建前端（可选）

```cmd
# 安装依赖
pnpm install

# 构建
pnpm run build
```

## Windows 服务

### 使用 NSSM 安装为 Windows 服务

1. 下载 NSSM: https://nssm.cc/download

2. 安装服务：
   ```cmd
   nssm install BigDataConfigGenerator "C:\path\to\project\server.exe" --web

   # 配置服务
   nssm set BigDataConfigGenerator AppDirectory "C:\path\to\project"
   nssm set BigDataConfigGenerator DisplayName "BigData Config Generator"
   nssm set BigDataConfigGenerator Description "大数据平台配置生成器"
   nssm set BigDataConfigGenerator Start SERVICE_AUTO_START
   ```

3. 启动服务：
   ```cmd
   nssm start BigDataConfigGenerator
   ```

4. 查看日志：
   ```cmd
   # 查看服务状态
   nssm status BigDataConfigGenerator

   # 编辑日志输出路径
   nssm edit BigDataConfigGenerator
   ```

## 性能优化

### 1. 使用 SSD

将项目目录放在 SSD 上，提升 I/O 性能

### 2. 调整配置

减少不必要的节点和服务，提升生成速度

### 3. 增加内存

如果生成大量配置，确保系统有足够内存

## 备份与恢复

### 备份

```cmd
# 备份配置
copy config.yaml config.yaml.backup

# 备份模板
xcopy /E /I templates templates.backup

# 备份输出
xcopy /E /I output output.backup
```

### 恢复

```cmd
# 恢复配置
copy config.yaml.backup config.yaml

# 恢复模板
xcopy /E /I templates.backup templates
```

## 日志与调试

### 查看日志

程序日志会输出到控制台。

### 调试模式

如需更详细的日志，可以通过环境变量：

```cmd
set DEBUG=1
server.exe --web
```

## 卸载

```cmd
# 停止服务（如果安装为 Windows 服务）
nssm stop BigDataConfigGenerator
nssm remove BigDataConfigGenerator confirm

# 删除文件
rd /s /q your-project
```

## 获取帮助

如遇到问题：

1. 查看启动日志中的错误信息
2. 检查文件权限
3. 确认目录结构正确
4. 参考 README.md 和 USAGE_GUIDE.md
