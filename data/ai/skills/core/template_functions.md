模板函数参考。

仅使用项目已注册函数，不要臆造额外 helper。
若某个函数无法满足需求，请改写模板逻辑，并在 warnings 说明。

1. join
- 作用：把数组拼接成字符串
- 正确示例：`{{ join "," (serviceHostnames "zookeeper") }}`

2. split
- 作用：把字符串按分隔符拆分
- 正确示例：`{{ split "," "a,b,c" }}`

3. default
- 作用：当值为空或 nil 时返回默认值
- 正确示例：`{{ default "2181" (serviceVar "zookeeper" "client_port") }}`

4. toUpper
- 作用：转大写
- 正确示例：`{{ toUpper .Instance.ServiceName }}`

5. toLower
- 作用：转小写
- 正确示例：`{{ toLower .Instance.ServiceName }}`

6. trim
- 作用：去掉字符串首尾空白
- 正确示例：`{{ trim .Instance.Vars.cluster_name }}`

7. replace
- 作用：替换字符串内容
- 正确示例：`{{ replace .Instance.Node.Hostname "." "-" }}`

8. add
- 作用：整数加法
- 正确示例：`{{ add 1 2 }}`

9. sub
- 作用：整数减法
- 正确示例：`{{ sub 10 1 }}`

10. mul
- 作用：整数乘法
- 正确示例：`{{ mul 2 3 }}`

11. div
- 作用：整数除法
- 正确示例：`{{ div 8 2 }}`

12. serviceNodes
- 作用：返回某个服务的全部实例，适合 range 遍历
- 正确示例：`{{ range serviceNodes "zookeeper" }}{{ .Node.IP }} {{ end }}`

13. serviceEndpoints
- 作用：按服务名和端口字段返回 ip:port 列表
- 正确示例：`{{ join "," (serviceEndpoints "zookeeper" "client_port") }}`

14. serviceEndpointsJoin
- 作用：直接返回逗号拼接后的 endpoints 字符串
- 正确示例：`{{ serviceEndpointsJoin "zookeeper" "client_port" }}`

15. serviceIPs
- 作用：返回某个服务的 IP 列表
- 正确示例：`{{ join "," (serviceIPs "elasticsearch") }}`

16. serviceHostnames
- 作用：返回某个服务的主机名列表
- 正确示例：`{{ join "," (serviceHostnames "elasticsearch") }}`

17. getServiceNodes
- 作用：返回某个服务涉及到的节点别名列表
- 正确示例：`{{ join "," (getServiceNodes "hdfs_namenode") }}`

18. serviceVars
- 作用：返回某个服务的全部 vars map
- 正确示例：`{{ index (serviceVars "zookeeper") "client_port" }}`

19. serviceVar
- 作用：读取某个服务的单个 vars 值
- 正确示例：`{{ serviceVar "zookeeper" "client_port" }}`
- 注意：当前只有两个参数，不支持第三个默认值参数
- 错误示例：`{{ serviceVar "zookeeper" "client_port" "2181" }}`

20. serviceConfig
- 作用：返回某个服务的完整 serverConfig 对象
- 正确示例：`{{ (serviceConfig "zookeeper").Description }}`

21. nodeInfo
- 作用：按节点别名返回 nodes.<nodeName> 节点信息
- 正确示例：`{{ with nodeInfo "master1" }}{{ .IP }} {{ .Hostname }}{{ end }}`

使用原则：
- 集群地址、节点列表优先使用 service* 系列函数
- 默认值优先使用 default
- 不要使用未注册函数
