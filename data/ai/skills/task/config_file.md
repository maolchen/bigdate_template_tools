用于服务配置文件模板。

硬约束：
- 环境相关值优先参数化到 `.Global` 和 `.Instance.Vars`
- 能参数化的 host/IP/user/group/path/port 不要硬编码
- 文件命名应与真实配置文件名一致
- 默认按同目录输出理解，配置文件会和脚本一起出现在 `<Node.IP>/<service>/`
- 除非用户明确要求，否则不要人为增加中间目录层级

推荐写法：
- 单节点值：`{{ .Instance.Node.IP }}`、`{{ .Instance.Node.Hostname }}`
- 服务目录：`{{ .Global.install_base_dir }}/{{ .Instance.Vars.install_subdir }}`
- 集群端点：`{{ join "," (serviceEndpoints "zookeeper" "client_port") }}`
- 兜底值：`{{ default "value" .Instance.Vars.some_key }}`

适用场景：
- 主配置文件
- 环境变量文件
- properties / yaml / conf / xml / ini 等配置模板

避免：
- 写死集群节点列表
- 写死部署目录
- 写死用户和组
- 使用错误历史变量名
