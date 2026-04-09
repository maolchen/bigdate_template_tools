# AI 模板生成全局规则

## 1. 目标与边界

- 目标是为本项目生成可审核、可保存、可渲染的 `templates/**/*.tmpl` 草稿。
- AI 输出只能是草稿、计划动作、配置补丁建议，不能假设已经写入真实文件。
- 不允许新增节点。`serviceTop.nodes` 只能从当前已配置的节点别名中选择，或使用 `["*"]`。
- 新服务模板通常需要同时补齐：
  - `serviceTop.<service>`
  - `serverConfig.<service>.vars`

## 2. 输出约束

- 模板路径必须位于 `templates/<service>/...` 下，且必须以 `.tmpl` 结尾。
- 安装脚本优先命名为：
  - `install.sh.tmpl`
  - `install_binary.sh.tmpl`
  - `setup_dirs.sh.tmpl`
  - `start.sh.tmpl`
  - `stop.sh.tmpl`
  - `check.sh.tmpl`
- 配置文件模板按真实文件名命名，例如：
  - `my.cnf.tmpl`
  - `docker.service.tmpl`
  - `dnsmasq.conf.tmpl`
- 如果需要 README，只在用户明确要求时生成 `README.md.tmpl`。

## 3. 全局变量优先级

- 路径、用户、组、JDK 等公共信息优先复用 `.Global`，不要在服务级变量中重复定义。
- 必须优先使用这些全局变量：
  - `{{ .Global.install_base_dir }}`
  - `{{ .Global.data_base_dir }}`
  - `{{ .Global.software_dir }}`
  - `{{ .Global.temp_dir }}`
  - `{{ .Global.java_home }}`
  - `{{ .Global.user }}`
  - `{{ .Global.group }}`
- 不要新增这些重复变量到 `serverConfig.vars`：
  - `run_user`
  - `run_group`
  - `java_home`
  - `install_base_dir`
  - `data_base_dir`

## 4. 服务变量设计规则

- 服务安装目录统一使用：
  - `install_subdir`
- 服务数据目录统一使用：
  - `data_subdir`
- 安装包相对目录优先使用：
  - `package_subdir`
- 版本号统一使用：
  - `version`
- 如果是集群组件，可以按功能分层组织变量，允许使用嵌套 vars，例如：
  - `zookeeper.client_port`
  - `zookeeper.data_subdir`
  - `server.port`
  - `server.log_subdir`
- 如果模板里引用了 `.Instance.Vars.xxx`，就必须同时给出对应的 `serverConfig.vars.xxx` 草案。
- 未知变量可以留空字符串占位，但必须在说明里提示人工补齐。

## 5. 路径拼接规则

- 数据目录统一拼接方式：
  - `{{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}`
- 安装目录统一拼接方式：
  - `{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}`
- 软件包目录统一拼接方式：
  - `{{ .Global.software_dir }}/{{ .Instance.Vars.package_subdir }}`
- 临时目录优先使用：
  - `{{ .Global.temp_dir }}`
- 除非服务本身有强约束，不要硬编码绝对目录。
- 如果服务必须落到固定路径，如 `/usr/local`、`/etc/systemd/system`、`/usr/bin`，必须在 reason 或 warning 中注明原因。

## 6. Shell 模板规则

- 安装脚本默认假设以 `{{ .Global.user }}` 作为主要运维用户。
- 所有需要 root 权限的命令都应通过 `run_as_root()` 包装。
- 优先使用如下辅助函数结构：

```bash
run_as_root() {
  if [ "$(whoami)" != "root" ]; then
    sudo "$@"
  else
    "$@"
  fi
}
```

- 非必要不要直接使用 root 启动服务。
- 使用 systemd 时，`User` 和 `Group` 优先使用：
  - `{{ .Global.user }}`
  - `{{ .Global.group }}`
- shell 脚本必须尽量幂等：
  - 目录使用 `mkdir -p`
  - 软链接先删后建
  - 配置覆盖前先备份
  - 重复执行不应报错或产生脏状态
- 推荐包含基础日志函数，例如：
  - `log_info`
  - `log_warn`
  - `log_error`
  - `log_success`

## 7. 环境变量与 profile 规则

- 环境变量配置要考虑 `{{ .Global.user }}` 是否为 root：
  - root 用户可写 `/etc/profile` 或 `/etc/profile.d/*.sh`
  - 非 root 用户优先写用户 profile，必要时再补系统 profile
- `JAVA_HOME` 必须优先使用 `{{ .Global.java_home }}`

## 8. 集群与模板函数规则

- 生成集群服务模板时，优先使用项目已有模板函数，而不是手工硬编码节点列表。
- 优先考虑这些模板函数：
  - `serviceNodes`
  - `serviceEndpoints`
  - `serviceEndpointsJoin`
  - `serviceIPs`
  - `serviceHostnames`
  - `getServiceNodes`
  - `serviceVars`
  - `serviceVar`
  - `serviceConfig`
  - `nodeInfo`
- 可用上下文：
  - `.Global`
  - `.Nodes`
  - `.Instance`
  - `.AllInstances`
- 如果模板需要实例唯一标识，优先使用：
  - `{{ .Instance.AutoID }}`

## 9. serviceTop 补丁规则

- 新服务模板通常必须给出 `serviceTop` 草案。
- `serviceTop` 只包含与模板直接相关的服务。
- `serviceTop.<service>.nodes` 必须：
  - 使用已配置节点别名
  - 或使用 `["*"]`
- 如果用户要求“3 节点集群”，但当前配置节点不足 3 个，必须在 `followUpQuestions` 里要求用户确认，不要编造节点。
- `id_auto_derive` 只有在实例 ID 明显应由节点顺序推导时才设为 `true`。

## 10. serverConfig 补丁规则

- 新服务模板通常必须给出 `serverConfig.<service>` 草案。
- `description` 要简洁说明服务用途。
- `vars` 只给模板真实需要的变量，不要把无关变量全部塞进去。
- 已有服务如需扩展模板，可只补充缺失变量，不要无理由覆盖现有变量。

## 11. 参考现有模板的风格

- 当前仓库已有这些典型模板风格，应尽量对齐：
  - `templates/kafka/install.sh.tmpl`
  - `templates/hadoop3/setup_dirs.sh.tmpl`
  - `templates/docker/install.sh.tmpl`
  - `templates/mysql/my.cnf.tmpl`
  - `templates/dnsmasq/dnsmasq.conf.tmpl`
- 对齐点包括：
  - 注释头结构清晰
  - 变量区集中定义
  - 辅助函数先定义再调用
  - 目录、权限、systemd、配置生成分步骤组织

## 12. 特殊服务处理

- 某些服务允许固定路径或系统目录写入，例如：
  - Docker
  - systemd service 文件
  - 部分数据库或系统服务
- 遇到这类服务时：
  - 可以写固定路径
  - 但必须明确说明为什么不能完全走全局目录
  - 仍要保证 `{{ .Global.user }}` 有可运维权限

## 13. 图片与文本改写规则

- 图片只用于识别文字截图、脚本截图、配置截图、文档拍照。
- 对识别不清、遮挡、低分辨率、截图不完整的部分，必须写入 `warnings`。
- 从文本或图片改写成模板时：
  - 优先抽取可参数化部分
  - 把环境相关值改写为 `.Global` 或 `.Instance.Vars`
  - 保留脚本执行顺序和关键语义

## 14. 禁止事项

- 不要新增节点定义。
- 不要把用户未确认的信息写成确定配置。
- 不要生成无法通过 Go Template 解析的语法。
- 不要把模板写成只能执行一次的非幂等脚本。
- 不要无理由硬编码 IP、主机名、用户、组、JDK 路径、安装根目录、数据根目录。

## 15. 默认工作方式

- 先判断用户要的是：
  - 新建目录
  - 新建模板
  - 改写现有脚本
  - 改写配置文件
- 如果是新服务：
  - 生成目录计划
  - 生成模板草稿
  - 生成 `configPatch`
- 如果是已有服务补模板：
  - 只补充相关模板
  - 只补充必要的 `vars` 或 `serviceTop` 调整
