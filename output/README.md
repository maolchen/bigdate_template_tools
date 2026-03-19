# 大数据平台部署总览

## 节点列表

| 节点名 | IP | 主机名 |
|---|---|---|
| node1 | 192.168.10.10 | master01 |
| node2 | 192.168.10.11 | master02 |
| node3 | 192.168.10.12 | master03 |
| node4 | 192.168.10.13 | worker01 |
| node5 | 192.168.10.14 | worker02 |
| node6 | 192.168.10.15 | worker03 |
| node7 | 192.168.10.16 | es01 |
| node8 | 192.168.10.17 | es02 |
| node9 | 192.168.10.18 | es03 |

## 服务拓扑

| 服务名 | 节点数 | 部署节点 | 自动ID |
|---|---|---|---|
| dolphinscheduler_api | 1 | node1 | 否 |
| dolphinscheduler_master | 1 | node1 | 否 |
| dolphinscheduler_worker | 4 | node3, node4, node5, node6 | 否 |
| elasticsearch | 3 | node7, node8, node9 | 否 |
| flink_jobmanager | 1 | node1 | 否 |
| flink_taskmanager | 4 | node3, node4, node5, node6 | 否 |
| hadoop_datanode | 4 | node3, node4, node5, node6 | 否 |
| hadoop_namenode | 2 | node1, node2 | ✅ |
| kafka | 3 | node4, node5, node6 | ✅ |
| spark_master | 1 | node1 | 否 |
| spark_worker | 4 | node3, node4, node5, node6 | 否 |
| zookeeper | 3 | node1, node2, node3 | ✅ |

## 主机部署明细

| IP | 主机名 | 部署服务 |
|---|---|---|
| 192.168.10.10 | master01 | spark_master, flink_jobmanager, dolphinscheduler_master, zookeeper, hadoop_namenode, dolphinscheduler_api |
| 192.168.10.11 | master02 | zookeeper, hadoop_namenode |
| 192.168.10.12 | master03 | hadoop_datanode, dolphinscheduler_worker, zookeeper, spark_worker, flink_taskmanager |
| 192.168.10.13 | worker01 | kafka, hadoop_datanode, dolphinscheduler_worker, spark_worker, flink_taskmanager |
| 192.168.10.14 | worker02 | kafka, hadoop_datanode, dolphinscheduler_worker, spark_worker, flink_taskmanager |
| 192.168.10.15 | worker03 | kafka, hadoop_datanode, dolphinscheduler_worker, spark_worker, flink_taskmanager |
| 192.168.10.16 | es01 | elasticsearch |
| 192.168.10.17 | es02 | elasticsearch |
| 192.168.10.18 | es03 | elasticsearch |

## 交付步骤

1. 将 `output/<ip>` 目录上传到对应主机
2. 在每台主机上执行各服务的 `install.sh`
3. 启动服务
