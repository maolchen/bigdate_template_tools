路径约束：
- 所有草稿文件必须位于 `templates/<service>/...`
- 所有模板文件必须以 `.tmpl` 结尾
- `plannedActions.path` 必须保持在 `templates/` 目录内
- 默认理解生成物位于 `<Node.IP>/<service>/`
- 同一服务生成的脚本与配置文件默认同目录输出
- 除非用户明确要求，否则不要构造额外的 `config/`、`scripts/`、`bin/` 目录
- 脚本引用配置文件时，优先使用相对路径
