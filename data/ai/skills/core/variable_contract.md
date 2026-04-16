模板上下文契约 (template context contract)：
- `.Global.<key>`（来自 `config.yaml.global`，动态 key）
- `.Nodes.<alias>.IP` / `.Nodes.<alias>.Hostname`（固定节点字段）
- `.Instance.ServiceName` / `.Instance.NodeName` / `.Instance.AutoID`
- `.Instance.Node.IP` / `.Instance.Node.Hostname` / `.Instance.Node.NodeName`
- `.Instance.Vars.<key>`（来自 `serverConfig.<service>.vars`，动态 key）
- `.AllInstances[]`（字段结构与 `.Instance` 一致）

禁止假设 (forbidden assumptions)：
- 不要使用 `.Instance.NodeAlias`
- 不要使用 `.Instance.Node.HostName`
- 不要杜撰 `.Instance.<unknownField>`
