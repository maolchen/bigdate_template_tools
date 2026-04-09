Docker 特殊规则：
- 允许涉及 /etc/docker、/usr/lib/systemd/system、docker.service 这类系统路径。
- 但仍需优先复用全局用户、组和软件目录，并明确说明哪些路径是 Docker 的固定系统约束。
