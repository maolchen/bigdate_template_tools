用于 systemd unit 模板 (for systemd unit templates)：
- 优先使用 `{{ .Global.user }}` 和 `{{ .Global.group }}`
- unit 文件保持聚焦，不要混入完整安装逻辑
- 路径优先通过 `.Global` 与 `.Instance.Vars` 参数化
