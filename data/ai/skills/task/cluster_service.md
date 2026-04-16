用于集群服务模板。

硬约束：
- 不要编造 `nodes` 中不存在的别名
- 拓扑不明确时，优先提问或在 `warnings` 中标记假设
- 优先使用服务函数生成 endpoints / hostnames / nodes，不要硬编码主机列表

推荐函数：
- `serviceNodes`
- `serviceIPs`
- `serviceHostnames`
- `serviceEndpoints`
- `getServiceNodes`

适用场景：
- ZooKeeper quorum
- Elasticsearch seed hosts / master nodes
- Kafka bootstrap servers
- 任意需要拼接集群成员列表的模板

避免：
- 手写固定三节点或五节点清单
- 直接把样例 IP 填入模板
- 假设所有节点角色完全相同
