模板契约：

文件与路径：
- 草稿文件路径必须位于 `templates/<service>/`
- 文件后缀必须为 `.tmpl`
- 非特殊说明下，模板生成物默认位于 `<Node.IP>/<service>/` 同级目录
- 因此配置文件与脚本默认可使用相对路径协作
- 除非用户明确指定，不要额外创建 `config/`、`bin/`、`scripts/` 等子目录

服务模板组织：
- 一个服务默认至少只有一个 `install.sh.tmpl`
- 若用户没有明确要求，不要把安装步骤拆成多个 shell 脚本
- 启动/停止/检查脚本是否生成，取决于服务是否真的需要
- 若服务只需安装和配置，不要为了“完整性”硬生成无用脚本

变量契约：
- 节点主机名必须使用 `{{ .Instance.Node.Hostname }}`
- 节点 IP 必须使用 `{{ .Instance.Node.IP }}`
- 节点别名可使用 `{{ .Instance.NodeName }}` 或 `{{ .Instance.Node.NodeName }}`
- 禁止使用 `.Instance.NodeAlias`
- 禁止使用 `.Instance.Node.HostName`

安装契约：
- 默认按二进制包安装理解
- 安装包路径、安装目录、数据目录、日志目录应参数化
- 非必要不使用 root
- 仅在需要写系统目录、改权限、装 systemd、改 `/etc/*` 时使用 `run_as_root`

冲突处理：
- 若历史草稿、用户提示或旧示例与本契约冲突，优先遵守本契约
- 若存在无法确定的冲突，在 `warnings` 中点明，不要自行猜测

显式调用：
- `#skill:core/template_contract`
- `@skill(core/template_contract)`
