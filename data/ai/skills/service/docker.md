Docker 模板补充约束 (Docker-specific guidance)：
- 允许使用系统路径（如 `/etc/docker`、systemd 路径）
- user/group 和通用目录仍优先来自 `.Global`
- 服务参数优先来自 `.Instance.Vars`
