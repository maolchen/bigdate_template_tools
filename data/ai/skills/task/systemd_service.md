用于 systemd unit 模板。

硬约束：
- 优先使用 `{{ .Global.user }}` 和 `{{ .Global.group }}`
- unit 文件只描述服务启动行为，不要混入完整安装逻辑
- `ExecStart`、`ExecStop`、`WorkingDirectory` 等路径优先通过 `.Global` 与 `.Instance.Vars` 参数化

适用前提：
- 只有服务确实需要 systemd 管理时才生成
- 若服务只是一次性初始化脚本或简单工具，不要强行生成 unit 文件

推荐：
- 配合 `install.sh.tmpl` 完成 unit 拷贝、权限调整、`daemon-reload` 和 `enable`
- 服务尽量使用非 root 用户运行
- 若确需 root 运行，应明确原因
