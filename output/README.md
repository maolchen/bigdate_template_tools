# 大数据平台部署总览

生成时间: 2025-01-XX

## 全局变量

| 变量名 | 值 |
|---|---|
| jdk_path | /data/jdk |
| java_home | /data/jdk/jdk1.8.0_65 |
| temp_dir | /data/tmp_install_dir |
| jdk_version | 1.8.0_65 |
| install_base_dir | /data/localization |
| data_base_dir | /data/bigdata |
| log_base_dir | /var/log/bigdata |
| user | bigdata |
| group | bigdata |
| software_dir | /data/softwares |
| timezone | Asia/Shanghai |

## 集群拓扑

| IP | 主机名 | 部署服务 |
|---|---|---|
| 192.168.10.12 | node3 | spark_worker, zookeeper, flink_taskmanager, dolphinscheduler_worker, hadoop_datanode |
| 192.168.10.10 | node1 | flink_jobmanager, zookeeper, dolphinscheduler_master, dolphinscheduler_api, spark_master, hadoop_namenode |
| 192.168.10.16 | node7 | elasticsearch |
| 192.168.10.13 | node4 | kafka, spark_worker, flink_taskmanager, dolphinscheduler_worker, hadoop_datanode |
| 192.168.10.14 | node5 | kafka, spark_worker, flink_taskmanager, dolphinscheduler_worker, hadoop_datanode |
| 192.168.10.15 | node6 | kafka, spark_worker, flink_taskmanager, dolphinscheduler_worker, hadoop_datanode |
| 192.168.10.11 | node2 | zookeeper, hadoop_namenode |
| 192.168.10.17 | node8 | elasticsearch |
| 192.168.10.18 | node9 | elasticsearch |

## 服务统计

| 服务名 | 节点数 | 节点列表 |
|---|---|---|
| spark_master | 1 | 192.168.10.10 |
| hadoop_datanode | 4 | 192.168.10.12, 192.168.10.13, 192.168.10.14, 192.168.10.15 |
| hadoop_namenode | 2 | 192.168.10.10, 192.168.10.11 |
| dolphinscheduler_api | 1 | 192.168.10.10 |
| flink_taskmanager | 4 | 192.168.10.13, 192.168.10.14, 192.168.10.15, 192.168.10.12 |
| dolphinscheduler_worker | 4 | 192.168.10.15, 192.168.10.12, 192.168.10.13, 192.168.10.14 |
| kafka | 3 | 192.168.10.13, 192.168.10.14, 192.168.10.15 |
| spark_worker | 4 | 192.168.10.13, 192.168.10.14, 192.168.10.15, 192.168.10.12 |
| flink_jobmanager | 1 | 192.168.10.10 |
| zookeeper | 3 | 192.168.10.10, 192.168.10.11, 192.168.10.12 |
| dolphinscheduler_master | 1 | 192.168.10.10 |
| elasticsearch | 3 | 192.168.10.17, 192.168.10.18, 192.168.10.16 |
