# 模板规范检查与修改记录

## 检查日期
2024-03-25

## 规范检查结果

### 符合规范的模板 ✅

以下模板已正确使用全局变量：

| 模板 | 用户/组 | JAVA_HOME | 路径拼接 | 状态 |
|------|---------|-----------|----------|------|
| hadoop_namenode/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| hadoop_journalnode/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| hadoop_datanode/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| hadoop_zkfc/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| hadoop_resourcemanager/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| hadoop_nodemanager/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| kafka/install.sh.tmpl | ✅ | ✅ | ✅ | 已修复 |
| redis/install.sh.tmpl | ✅ | N/A | ✅ | 待修复 |
| ssdb/install.sh.tmpl | ✅ | N/A | ✅ | 待修复 |
| docker/install.sh.tmpl | ✅ | N/A | ✅ | 待修复 |
| mysql/install.sh.tmpl | ✅ | N/A | ✅ | 待修复 |

### 不符合规范的问题 ❌

#### 问题1: 环境变量配置不规范

**规范要求**:
> 环境变量的配置，根据 `{{ .Global.user }}` 判断是否为root用户，如果是root用户，则配置到 `/etc/profile`，如果是非root用户，则配置到 `~/.bash_profile` 和 `/etc/profile`

**当前问题**:
所有模板直接配置到 `/etc/profile.d/` 或 `/etc/profile`，未根据用户判断。

**影响模板**:
- hadoop_namenode/install.sh.tmpl ✅ 已修复
- kafka/install.sh.tmpl ✅ 已修复
- 其他需要配置环境变量的模板

#### 问题2: 脚本编写规范不完整

**规范要求**:
> 所有的shell脚本，默认使用 `{{ .Global.user }}` 这个用户进行执行，如果该用户为非root用户，所有需要进行root用户的操作，均在脚本中使用sudo提权。

**当前问题**:
- 脚本未添加用户判断逻辑
- 未在需要root权限的操作前添加sudo

**影响模板**:
- 所有install.sh.tmpl模板

#### 问题3: 服务部署和启动规范

**规范要求**:
> 所有的服务非必要不使用root用户启动，使用 `{{ .Global.user }}` 启动，systemctl管理的或者有特殊说明的除外

**当前问题**:
- 部分服务使用root启动
- systemd服务配置中User字段未正确使用全局变量

**影响模板**:
- kafka/install.sh.tmpl (systemd服务配置) ✅ 已修复
- 其他使用systemd的服务

---

## 已完成的修改

### 2024-03-25

#### 1. hadoop_namenode/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑 `run_as_root()` 函数
- 环境变量配置根据用户类型选择配置文件（root: /etc/profile.d/, 非root: ~/.bash_profile + /etc/profile）
- 所有需要root权限的操作使用 `run_as_root` 函数

#### 2. hadoop_journalnode/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑
- 目录创建使用 `run_as_root` 函数
- 权限设置使用 `run_as_root` 函数

#### 3. hadoop_datanode/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑
- 目录创建使用 `run_as_root` 函数
- 权限设置使用 `run_as_root` 函数

#### 4. hadoop_zkfc/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑
- 目录创建使用 `run_as_root` 函数

#### 5. hadoop_resourcemanager/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑
- 环境变量配置根据用户类型选择配置文件
- 所有需要root权限的操作使用 `run_as_root` 函数

#### 6. hadoop_nodemanager/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑
- 目录创建使用 `run_as_root` 函数
- 权限设置使用 `run_as_root` 函数

#### 7. kafka/install.sh.tmpl
**修改内容**:
- 添加用户权限判断逻辑 `run_as_root()` 函数
- 修复systemd服务配置，使用 `{{ .Global.user }}` 和 `{{ .Global.group }}`
- 修复JAVA_HOME配置，使用 `{{ .Global.java_home }}`
- 目录创建使用 `run_as_root` 函数
- 权限设置使用 `run_as_root` 函数

---

## 待修复模板列表

- [ ] redis/install.sh.tmpl
- [ ] ssdb/install.sh.tmpl
- [ ] docker/install.sh.tmpl
- [ ] mysql/install.sh.tmpl
- [ ] 其他模板
