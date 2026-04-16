Elasticsearch 专项约束 (Elasticsearch-specific constraints)：
- 节点占位符必须是 `{{ .Instance.Node.Hostname }}`
- 禁止 `.Instance.NodeAlias` 与 `.Instance.Node.HostName`
- 默认保持 `templates/elasticsearch/*.tmpl` 平铺
- 除非用户明确要求，不要放到 `templates/elasticsearch/config/`
- 若用户要求“修改 Elasticsearch 全部模板”，必须覆盖相关全部草稿，而不是只改一个文件

自动触发关键词 (keywords to auto attach)：
- `elasticsearch`
- `nodealias`
- `output/<ip>/<service>`
- `全部模板`
- `不要 config 子目录`
