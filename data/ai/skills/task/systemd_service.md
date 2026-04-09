systemd 模板规则：
- systemd unit 文件通常属于配置模板，不要把整套安装逻辑写进 unit。
- User 和 Group 优先使用 {{ .Global.user }} 和 {{ .Global.group }}。
- 如果 unit 文件需要引用脚本或目录，优先使用全局目录和实例变量拼接。
- 涉及 /etc/systemd/system 这类固定系统路径时，要在说明里标记为系统目录约束。
