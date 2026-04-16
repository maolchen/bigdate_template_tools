模板契约 (strict template contract)：
- 节点主机名占位符必须使用 `{{ .Instance.Node.Hostname }}`
- 禁止占位符：`.Instance.NodeAlias`、`.Instance.Node.HostName`
- 模板路径必须在 `templates/<service>/` 下，且文件后缀为 `.tmpl`
- 默认输出结构是 `output/<ip>/<service>/` 平铺目录
- 除非用户明确要求嵌套目录，否则不要创建 `templates/<service>/config/...`
- 若旧草稿与本契约冲突，优先遵循本契约，并在 `warnings` 解释

显式调用 (explicit invocation)：
- `#skill:core/template_contract`
- `@skill(core/template_contract)`
