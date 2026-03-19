# 大数据平台部署总览

## 节点列表

| 节点名 | IP | 主机名 |
|---|---|---|
| realtime-dw1 | 192.168.10.10 | realtime-dw1 |
| realtime-dw2 | 192.168.10.11 | realtime-dw2 |
| realtime-dw3 | 192.168.10.12 | realtime-dw3 |
| realtime-es1 | 192.168.10.16 | realtime-es1 |
| realtime-es2 | 192.168.10.17 | realtime-es2 |
| realtime-es3 | 192.168.10.18 | realtime-es3 |
| realtime-kafka1 | 192.168.10.13 | realtime-kafka1 |
| realtime-kafka2 | 192.168.10.14 | realtime-kafka2 |
| realtime-kafka3 | 192.168.10.15 | realtime-kafka3 |

## 服务拓扑

| 服务名 | 节点数 | 部署节点 | 自动ID |
|---|---|---|---|
| dolphinscheduler_api | 1 | realtime-dw1 | 否 |
| dolphinscheduler_master | 1 | realtime-dw1 | 否 |
| dolphinscheduler_worker | 3 | realtime-dw1, realtime-dw2, realtime-dw3 | 否 |
| elasticsearch | 3 | realtime-es1, realtime-es2, realtime-es3 | 否 |
| flink_jobmanager | 1 | realtime-dw1 | 否 |
| flink_taskmanager | 3 | realtime-dw1, realtime-dw2, realtime-dw3 | 否 |
| hadoop_datanode | 3 | realtime-dw1, realtime-dw2, realtime-dw3 | 否 |
| hadoop_namenode | 2 | realtime-dw1, realtime-dw2 | ✅ |
| kafka | 3 | realtime-kafka1, realtime-kafka2, realtime-kafka3 | ✅ |
| spark_master | 1 | realtime-dw1 | 否 |
| spark_worker | 3 | realtime-dw1, realtime-dw2, realtime-dw3 | 否 |
| zookeeper | 3 | realtime-dw1, realtime-dw2, realtime-dw3 | ✅ |

## 主机部署明细

| IP | 主机名 | 部署服务 |
|---|---|---|
| 192.168.10.10 | realtime-dw1 | zookeeper, flink_jobmanager, dolphinscheduler_master, hadoop_namenode, hadoop_datanode, spark_master, spark_worker, flink_taskmanager, dolphinscheduler_worker, dolphinscheduler_api |
| 192.168.10.11 | realtime-dw2 | zookeeper, hadoop_namenode, hadoop_datanode, spark_worker, flink_taskmanager, dolphinscheduler_worker |
| 192.168.10.12 | realtime-dw3 | zookeeper, hadoop_datanode, spark_worker, flink_taskmanager, dolphinscheduler_worker |
| 192.168.10.13 | realtime-kafka1 | kafka |
| 192.168.10.14 | realtime-kafka2 | kafka |
| 192.168.10.15 | realtime-kafka3 | kafka |
| 192.168.10.16 | realtime-es1 | elasticsearch |
| 192.168.10.17 | realtime-es2 | elasticsearch |
| 192.168.10.18 | realtime-es3 | elasticsearch |

## 交付步骤

1. 将 `output/<ip>` 目录上传到对应主机
2. 在每台主机上执行各服务的 `install.sh`
3. 启动服务
