# AI 模板使用规范

本文档用于规范 `AI模板` 页面生成、修改、审核和保存 `templates/` 模板的使用方式，目标是减少 AI 杜撰变量、路径错误、配置同步不完整等问题。

## 1. 使用边界

`AI模板` 适合做：

- 新服务模板初稿生成。
- 已有模板批量改写。
- 根据安装包、官方文档、历史脚本生成安装/启动/停止/检查脚本。
- 根据模板变量生成 `serverConfig.vars` 建议。
- 根据服务部署方式生成 `serviceTop` 建议。

`AI模板` 不适合做：

- 不经人工审核直接发布生产模板。
- 直接修改普通用户的项目配置。
- 让 AI 猜测未知安装目录、包名、端口、账号密码。
- 一次性让 AI 生成过多服务，导致输出过长或审阅困难。

## 2. 权限规则

- 管理员可以生成草稿、编辑草稿、保存模板到 `templates/`。
- 普通用户可以生成草稿和查看内容，但不能保存到正式 `templates/`。
- `保存全部` / `保存当前` 只写模板文件。
- `保存并同步配置` 会尝试同步 AI 返回的配置补丁，但仍需人工确认 `serviceTop` 和 `serverConfig` 是否合理。

## 3. 推荐工作流

### 新服务模板

1. 准备服务信息：
   - 服务名，例如 `elasticsearch`、`spark3`。
   - 安装包路径，例如 `/data/software/elasticsearch-8.12.2.tar.gz`。
   - 安装目录，例如 `{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}`。
   - 数据目录、日志目录、配置文件目录。
   - 服务角色和节点部署方式。
   - 需要生成哪些模板文件。
2. 在提问框里使用明确提示词生成初稿。
3. 人工检查 AI 返回的模板草稿。
4. 在编辑模式中修改草稿。
5. 管理员执行 `保存全部` 或 `保存当前`。
6. 如需要配置补丁，再执行 `保存并同步配置`。
7. 到 `配置管理` 页面检查：
   - `serviceTop.<service>` 是否存在。
   - `serverConfig.<service>.vars` 是否完整。
   - 节点和变量值是否符合当前项目。
8. 到 `生成配置` 页面生成并检查输出文件。

### 修改已有服务模板

1. 明确修改范围，例如“修改 `templates/elasticsearch/` 下所有模板”。
2. 明确旧写法和新写法，例如：
   - 旧：`{{ .Instance.NodeAlias }}`
   - 新：`{{ .Instance.Node.Hostname }}`
3. 要求 AI 返回所有受影响文件的完整草稿，不要只返回解释。
4. 保存前逐个检查差异。
5. 保存后打开模板编辑器或文件系统确认实际文件已更新。

## 4. Skill 调用规范

系统支持通过 `#` 或显式 skill 名称引导 AI 使用模板约束。

推荐写法：

```text
#skill:core/template_contract
#skill:core/variable_contract
#skill:task/install_script
请为 xxx 服务生成模板。
```

也可以使用：

```text
@skill(core/template_contract)
@skill(core/variable_contract)
```

常用 skill 分类：

- `core/template_contract`：模板路径、禁止变量、输出目录结构约束。
- `core/variable_contract`：可用模板变量和禁止变量。
- `core/path_guard`：路径安全和目录约束。
- `core/output_json`：要求 AI 按系统可解析的 JSON 协议返回。
- `task/install_script`：安装脚本规范。
- `task/config_file`：配置文件模板规范。
- `task/config_patch`：配置补丁建议规范。
- `task/systemd_service`：systemd 服务模板规范。
- `task/cluster_service`：集群服务拓扑规划规范。
- `service/<service>`：某个服务的专用规则，例如 `service/elasticsearch`。

使用建议：

- 只是问普通问题时，不要调用 skill，避免 AI 输出模板协议内容。
- 生成或修改模板时，建议至少调用：
  - `core/template_contract`
  - `core/variable_contract`
  - 对应的 `task/*`
- 涉及具体服务时，再追加 `service/<service>`。

## 5. 模板变量规范

### 可动态扩展的变量

`.Global.<key>` 来自 `config.yaml.global`。

- 可以新增或删除 key。
- 新增后应同步到配置文件。
- 适合放全局复用变量，例如：
  - `user`
  - `group`
  - `install_base_dir`
  - `data_base_dir`
  - `log_base_dir`
  - `software_dir`
  - `java_home`

`.Instance.Vars.<key>` 来自 `serverConfig.<service>.vars`。

- 可以按服务新增或删除 key。
- 新增后应同步到 `serverConfig.<service>.vars`。
- 适合放服务专属变量，例如：
  - `install_subdir`
  - `data_subdir`
  - `log_subdir`
  - `http_port`
  - `cluster_name`
  - `heap_size`

### 固定变量

节点相关变量由代码结构固定，不能自定义字段名。

允许使用：

```gotemplate
{{ .Instance.Node.IP }}
{{ .Instance.Node.Hostname }}
{{ .Instance.Node.NodeName }}
{{ .Instance.NodeName }}
```

禁止使用：

```gotemplate
{{ .Instance.NodeAlias }}
{{ .Instance.Node.HostName }}
{{ .Instance.Hostname }}
{{ .Node.Hostname }}
```

### 实例变量

允许使用：

```gotemplate
{{ .Instance.ServiceName }}
{{ .Instance.AutoID }}
{{ .Instance.Vars.<key> }}
```

### 全局服务查询函数

常用函数包括：

```gotemplate
{{ serviceNodes "zookeeper" }}
{{ serviceIPs "zookeeper" }}
{{ serviceHostnames "zookeeper" }}
{{ serviceEndpoints "zookeeper" "client_port" }}
{{ serviceVar "zookeeper" "client_port" "2181" }}
{{ join "," (serviceHostnames "zookeeper") }}
{{ default "value" .Instance.Vars.some_key }}
```

实际可用函数以 `generator/template.go` 为准。

## 6. 路径与输出规范

模板文件路径必须在：

```text
templates/<service>/*.tmpl
```

默认生成输出路径为：

```text
output/users/<username>/<nodeIP>/<service>/
```

模板中不要假设生成文件会在额外的 `config/` 子目录下，除非用户明确要求。

例如，用户说明：

```text
install.sh.tmpl 和配置文件最终都在 output/<ip>/<service>/ 目录下。
```

则 AI 不应生成：

```text
output/<ip>/<service>/config/xxx.conf
```

## 7. 配置同步规则

`保存并同步配置` 只应处理 AI 返回的配置补丁建议，不代表配置一定完整正确。

### serviceTop

新服务必须有 `serviceTop.<service>`。

示例：

```yaml
serviceTop:
  elasticsearch:
    nodes:
      - node1
      - node2
      - node3
    description: Elasticsearch 集群节点
    id_auto_derive: false
```

如果 AI 没有给出 serviceTop，保存并同步配置时可能会提示阻塞，需要先让 AI 或人工补齐拓扑。

### serverConfig

新服务应有 `serverConfig.<service>`。

示例：

```yaml
serverConfig:
  elasticsearch:
    description: Elasticsearch 服务配置
    vars:
      install_subdir: elasticsearch
      data_subdir: elasticsearch/data
      log_subdir: elasticsearch/logs
      http_port: 9200
      transport_port: 9300
```

模板中新增 `.Instance.Vars.<key>` 后，应同步到 `serverConfig.<service>.vars`，否则生成时会出现空值或默认值不符合预期。

## 8. 推荐提示词模板

### 第一次生成服务模板

```text
#skill:core/template_contract
#skill:core/variable_contract
#skill:task/install_script
#skill:task/config_file
#skill:task/systemd_service

请为 <服务名> 生成一套可审核、可保存、可渲染的 Go Template 模板草稿。

服务部署信息：
- 服务名：<service>
- 版本：<version>
- 安装包路径：<安装包路径>
- 安装目录：使用 {{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}
- 数据目录：使用 {{ .Global.data_base_dir }}/{{ .Instance.Vars.data_subdir }}
- 日志目录：使用 {{ .Global.log_base_dir }}/{{ .Instance.Vars.log_subdir }}
- 运行用户/组：使用 {{ .Global.user }} / {{ .Global.group }}
- 当前节点主机名：必须使用 {{ .Instance.Node.Hostname }}
- 当前节点 IP：必须使用 {{ .Instance.Node.IP }}

部署要求：
- 安装脚本需要具备幂等性。
- 如果服务有配置文件，install.sh.tmpl 需要把生成后的配置文件覆盖或复制到服务实际配置目录。
- install.sh.tmpl 和配置文件最终都在 output/<ip>/<service>/ 目录下，不要额外假设 config 子目录。
- 需要给出 serviceTop 和 serverConfig.vars 的配置补丁建议。

请输出：
- 每个模板文件的 path、content、reason。
- serviceTop 建议。
- serverConfig.vars 建议。
- 如果某些参数无法确定，请放入 warnings，不要自行杜撰。
```

### 批量修改已有模板

```text
#skill:core/template_contract
#skill:core/variable_contract

请修改 templates/<service>/ 下所有模板：
- 将所有 {{ .Instance.NodeAlias }} 改为 {{ .Instance.Node.Hostname }}
- 将所有 {{ .Instance.Node.HostName }} 改为 {{ .Instance.Node.Hostname }}
- 不要修改无关逻辑
- 返回所有被修改文件的完整草稿
```

## 9. 人工审核清单

保存前至少检查：

- 模板路径是否都在 `templates/<service>/` 下。
- 文件名是否以 `.tmpl` 结尾。
- 是否存在禁止变量：
  - `.Instance.NodeAlias`
  - `.Instance.Node.HostName`
- 是否错误生成了多余目录，例如 `templates/<service>/config/...`。
- shell 脚本是否具备幂等性。
- 是否使用了全局用户和组：
  - `{{ .Global.user }}`
  - `{{ .Global.group }}`
- 是否所有 `.Instance.Vars.<key>` 都能在 `serverConfig.<service>.vars` 找到。
- `serviceTop` 是否包含正确节点。
- 配置文件是否会被安装脚本放到服务实际配置目录。
- 生成后输出目录结构是否符合 `output/<ip>/<service>/`。

## 10. 常见问题处理

### AI 使用了错误节点变量

现象：

```gotemplate
{{ .Instance.NodeAlias }}
```

处理：

- 要求 AI 使用 `core/variable_contract` 重新修改。
- 正确写法是：

```gotemplate
{{ .Instance.Node.Hostname }}
```

### 保存并同步配置提示缺少 serviceTop

原因：

- AI 只生成了模板文件，没有返回有效 `serviceTop` 补丁。
- 或者当前用户配置尚未加载最新保存后的服务拓扑。

处理：

- 到 `配置管理 > 服务拓扑` 手工补齐。
- 或继续让 AI 生成 `serviceTop` 配置补丁。
- 保存后刷新当前配置再重试。

### AI 返回纯文本，不能保存草稿

原因：

- 当前模型没有按系统 JSON 协议返回。
- 或 prompt 没有调用 `core/output_json`。

处理：

- 切回已验证支持结构化输出的模型。
- 或在 prompt 中加入：

```text
#skill:core/output_json
请严格按系统要求返回 JSON，不要输出额外解释文本。
```

### 一次输出被截断

处理：

- 按文件分批生成。
- 先生成安装脚本和配置文件，再生成启动/停止/检查脚本。
- 每轮明确“只返回这些文件”。

### 页面显示已修改但文件未更新

处理：

- 确认当前登录用户是管理员。
- 确认点击的是 `保存当前` 或 `保存全部`。
- 保存后打开 `模板编辑器` 或直接检查 `templates/<service>/`。
- 查看后端日志中的 `[AI] save write path=...`。

## 11. 管理员发布建议

管理员保存模板前建议执行：

```bash
pnpm --dir web build
go test ./server -run TestTemplateEditor
```

保存模板后建议执行：

```bash
go run main.go
```

或在 Web 页面执行一次 `生成配置`，确认输出文件路径和内容符合预期。

