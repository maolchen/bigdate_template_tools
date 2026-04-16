Elasticsearch 专项约束：
- 节点占位符必须使用 `{{ .Instance.Node.Hostname }}`
- 节点 IP 使用 `{{ .Instance.Node.IP }}`
- 禁止 `.Instance.NodeAlias` 与 `.Instance.Node.HostName`
- 默认保持 `templates/elasticsearch/*.tmpl` 平铺
- 除非用户明确要求，不要放到 `templates/elasticsearch/config/`
- Elasticsearch 默认按二进制包安装理解
- 若生成 `install.sh.tmpl`，优先在一个脚本里完成解压、目录准备、配置覆盖、权限处理、systemd 安装
- 若用户要求“修改 Elasticsearch 全部模板”，必须覆盖相关全部草稿，而不是只改一个文件

自动触发关键词：
- `elasticsearch`
- `nodealias`
- `output/<ip>/<service>`
- `全部模板`
- `不要 config 子目录`
