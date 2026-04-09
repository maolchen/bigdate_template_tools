配置文件模板规则：
- 优先把环境相关值参数化成 .Global 或 .Instance.Vars，不要硬编码 IP、主机名、目录、用户、JDK 路径。
- 配置模板命名尽量对齐真实文件名，例如 my.cnf.tmpl、dnsmasq.conf.tmpl、docker.service.tmpl。
- 允许保留服务本身要求的固定键名，但取值应参数化。
- 如果某些固定路径不可避免，必须在 reason 或 warnings 中说明原因。
