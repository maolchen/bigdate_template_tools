模板变量契约：

一、可动态扩展的变量
- `.Global.<key>`
  - 来源：`config.yaml > global`
  - 用于共享变量，如 user/group/java/path/software_dir
  - 可以新增和减少 key
- `.Instance.Vars.<key>`
  - 来源：`config.yaml > serverConfig.<service>.vars`
  - 用于服务变量，如版本、端口、目录、集群名
  - 可以新增和减少 key

二、固定结构变量
- `.Instance.ServiceName`
- `.Instance.NodeName`
- `.Instance.AutoID`
- `.Instance.Node.IP`
- `.Instance.Node.Hostname`
- `.Instance.Node.NodeName`
- `.AllInstances[]`
- `.Nodes`

三、固定结构字段不可自定义命名
- 不要使用 `.Instance.NodeAlias`
- 不要使用 `.Instance.Node.HostName`
- 不要使用 `.Instance.Hostname`
- 不要杜撰 `.Instance.<unknownField>`
- 不要杜撰 `.Node.<field>`

四、服务查询函数
- `serviceNodes "<service>"`
- `serviceIPs "<service>"`
- `serviceHostnames "<service>"`
- `serviceEndpoints "<service>" "<portField>"`
- `serviceVar "<service>" "<varName>"`
- `serviceVars "<service>"`
- `getServiceNodes "<service>"`
- `nodeInfo "<nodeName>"`
- `join "," (...)`
- `default "fallback" (...)`

五、使用约束
- 跨服务引用端口、IP、hostname 时，优先使用服务函数，不要手写节点清单
- 需要默认值时，用 `default`
- 当前 `serviceVar` 只有两个参数；不要写三参数版本

正确示例：
- `{{ .Global.user }}`
- `{{ .Instance.Node.Hostname }}`
- `{{ .Instance.Vars.install_subdir }}`
- `{{ join "," (serviceEndpoints "zookeeper" "client_port") }}`
- `{{ default "2181" (serviceVar "zookeeper" "client_port") }}`

错误示例：
- `{{ .Instance.NodeAlias }}`
- `{{ .Instance.Node.HostName }}`
- `{{ serviceVar "zookeeper" "client_port" "2181" }}`
