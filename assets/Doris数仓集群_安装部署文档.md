# Doris数仓集群安装部署

[本文档参考：Doris官方部署文档 - Apache Doris](https://doris.apache.org/zh-CN/docs/install/cluster-deployment/standard-deployment/)

##  使用非root用户部署时，以下操作需以普通用户操作。

## 1.软硬件需求
### 1.1 系统要求
* Linux操作系统版本需求  

  | Linux系统 | 版本 |
  | --- | --- |
  | CentOS | 7.x |
  | Kylin | v10_Spx |

* avx2指令集  
  当安装 Doris 时，建议选择支持 AVX2 指令集的机器，以利用 AVX2 的向量化能力实现查询向量化加速。  
  运行以下命令，有输出结果，及表示机器支持 AVX2 指令集。

  ```shell
  cat /proc/cpuinfo | grep avx2  # 如果命令输出为空，代表不支持avx2指令集
  ```
  ![doris_avx2](https://yunwei.zhugeio.com/docs_img/doris_avx2.png)

`注意：如果机器不支持 AVX2 指令集，部署时需要使用 no AVX2 的 Doris 安装包进行部署，否则无法运行。`

### 1.2 Doris部署安装包介绍
#### 1.2.1 Doris部署版本：v2.1.3
#### 安装包下载地址：
`文档中涉及安装包，可在基础安装包中找到：softwares-doris/realtime-new-doris/realtime-add/doris`
* 官方下载地址(仅做了解)  
[v2.1.3 标准版安装包下载](https://apache-doris-releases.oss-accelerate.aliyuncs.com/apache-doris-2.1.3-bin-x64.tar.gz)  
[v2.1.3 no-avx2版安装包下载](https://apache-doris-releases.oss-accelerate.aliyuncs.com/apache-doris-2.1.3-bin-x64-noavx2.tar.gz)


## 2.系统初始化or安装依赖 (所有集群节点)
`Doris在启动是会检查主机环境，如未达到要求不会启动`

### 2.1 关闭透明大页
在部署 Doris 时，建议关闭透明大页。

```shell
sudo sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/enabled'
sudo sh -c 'echo never > /sys/kernel/mm/transparent_hugepage/defrag'
```

### 2.2 设置系统最大打开文件句柄 【标准部署时如通过ansible已调整，此处可跳过】
```shell
sudo vim /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536
```

### 2.3 内核调整vm.max_map_count的大小
```shell
sudo sh -c 'echo "fs.file-max = 6553560" >> /etc/sysctl.conf'
sudo sh -c 'echo "vm.max_map_count=2000000" >> /etc/sysctl.conf'
sudo sysctl -p
```

### 2.4 关闭交换分区(swap)
linux 交换分区会给doris带来很严重的性能问题，需要再安装之前禁用交换分区

```plain
sudo swapoff -a     # 关闭交换分区
sudo swapon --show  # 确认交换分区已被关闭

sudo vim /etc/fstab
将swap挂载行 注释或删除
```
`如果要彻底禁用swap还需要修改内核启动参数：`

```plain

sudo yum -y install grubby
sudo grubby --info=ALL	#查看带swap的参数，然后用grubby删除swap相关的启动参数

sudo grubby --update-kernel=ALL --remove-args=resume=/dev/mapper/klas-swap	#args需要根据info=ALL实际查到的结果为准

sudo grubby --update-kernel=ALL --remove-args=rd.lvm.lv=klas/swap	#args需要根据info=ALL实际查到的结果为准

```



### 2.5 磁盘文件系统要求
```shell
ext4 / xfs  这两种文件系统均支持
```

### 2.6 gcc依赖安装
gcc版本需要满足4.8.2及以上
```shell
sudo yum install -y gcc-c++ bash-completion   
gcc -v # 验证版本
```

### 2.7 mysql客户端安装【标准部署时如通过ansible已安装，此处可跳过】
此客户端，便于doris启动后连接登录
```plain
sudo yum install mysql -y
 或
sudo yum install mariadb -y
```

### 2.8 jdk 【标准部署时如通过ansible已部署jdk，此处可跳过】
版本要求：jdk需要安装1.8及以上版本
```plain
wget http://47.92.192.89:88/realtime-2103/softwares-doris/realtime-new-doris/realtime/kafka/jdk-8u65-linux-x64.tar.gz
tar xvf jdk-8u65-linux-x64.tar.gz
mv jdk1.8.0_65/ /usr/local/jdk
####### 3.1.1.2 设置jdk环境变量
sudo vim /etc/profile
#JAVA
export JAVA_HOME=/usr/local/jdk
export JAVA_BIN=/usr/local/jdk/bin
export PATH=$PATH:$JAVA_HOME/bin
export CLASSPATH=.:$JAVA_HOME/lib/dt.jar:$JAVA_HOME/lib/tools.jar
export JAVA_HOME JAVA_BIN PATH CLASSPATH
####### 3.1.1.3 使环境变量及时生效
source /etc/profile
** 验证jdk有效性
java -version
```

### 2.9 ntp时钟同步 【标准部署时如通过ansible已部署ntp，此处可跳过】
```plain
Doris对集群内部时钟同步有要求，此处通过ntp服务保证集群各节点间时钟进行同步
Ps: 以其中一个FE节点为主，其他节点同步此节点时钟
```

## 3.集群规划[标准部署示例：1FE 3BE]
`Doris分为FE、BE等两个组件，FE为管理节点，BE为数据节点`  

| 主机 | 节点IP | 配置 | FE-Follower | BE | mysql-client |
| --- | --- | --- | --- | --- | --- |
| realtime-dw(realtime-web) | 10.0.0.5 | 8c/16G/200G | 安装 | | 安装 |
| realtime-dw1(realtime-1) | 10.0.0.6 | 8c/16G/200G | | 安装 | 安装 |
| realtime-dw2(realtime-2) | 10.0.0.7 | 8c/16G/200G | | 安装 | 安装 |
| realtime-dw3(realtime-3) | 10.0.0.9 | 8c/16G/200G | | 安装 | 安装 |

```shell
1.建议FE与BE分开独立部署
2.在生产环境中，如果 FE 与 BE 混布，需要注意资源争用问题，建议元数据存储与数据存储分盘存放
```

## 4.doris安装部署
### 4.1 下载安装（FE管理节点操作）
```shell
cd softwares-doris/realtime-new-doris/realtime-add/doris
tar xf apache-doris-2.1.3-bin-x64.tar.gz
mv apache-doris-2.1.3-bin-x64 /data/doris
```
```shell
如果使用no avx2版本包：apache-doris-2.1.3-bin-x64-noavx2.tar.gz
```

### 4.2 配置修改（FE管理节点操作）
#### 4.2.1 修改FE配置文件：/data/doris/fe/conf/fe.conf
```shell
# 端口相关
sed -i 's#8030#18030#g' /data/doris/fe/conf/fe.conf
sed -i 's#9020#19020#g' /data/doris/fe/conf/fe.conf
sed -i 's#9030#19030#g' /data/doris/fe/conf/fe.conf
sed -i 's#9010#19010#g' /data/doris/fe/conf/fe.conf

# 参数配置 【涉及网络和IP的参数请按实际环境修改】
echo "priority_networks = 172.16.0.0/24" >> /data/doris/fe/conf/fe.conf # 修改为当前集群网段
echo "meta_dir = /data/doris/doris-meta" >> /data/doris/fe/conf/fe.conf # fe的元数据存储目录
echo "max_dynamic_partition_num = 10000" >> /data/doris/fe/conf/fe.conf # 修改doris自动分区 
echo "max_allowed_packet = 33554432" >> /data/doris/fe/conf/fe.conf     # 调整最大数据包
echo "qe_max_connection = 102400" >> /data/doris/fe/conf/fe.conf        # 调整最大连接数
echo "max_connection_scheduler_threads_num = 409600" >> /data/doris/fe/conf/fe.conf #同上
echo "enable_outfile_to_local = true" >> /data/doris/fe/conf/fe.conf    # 查询结果导出文件支持
echo "table_name_length_limit = 256"  >> /data/doris/fe/conf/fe.conf    # 修改表名长度
```

#### 4.2.2 修改BE配置文件：/data/doris/be/conf/be.conf
```shell
# 端口相关
sed -i 's#9060#19060#g' /data/doris/be/conf/be.conf
sed -i 's#8040#18040#g' /data/doris/be/conf/be.conf
sed -i 's#9050#19050#g' /data/doris/be/conf/be.conf
sed -i 's#8060#18060#g' /data/doris/be/conf/be.conf
```
#### 4.2.3 修改broker端口（仅修改，Broker默认不会启动，数据迁移相关）
```shell
sed -i 's#8000#18000#g' /data/doris/extensions/apache_hdfs_broker/conf/apache_hdfs_broker.conf
```
#### 4.2.4 创建FE元数据目录
```shell
mkdir -p /data/doris/doris-meta
```

#### 4.2.5 创建udf目录
```shell
mkdir -p /data/doris/udf
```
#### 4.2.6 复制自定义udf包

`文档中涉及到的jar包，可在基础安装包中找到：softwares-doris/realtime-new-doris/realtime-add/doris`

```shell
cp softwares-doris/realtime-new-doris/realtime-add/doris/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar /data/doris/udf/
```

#### 4.2.7 复制jdbc_drivers

`文档中涉及到的jar包，可在基础安装包中找到：softwares-doris/realtime-new-doris/realtime-add/doris`

```shell
cp -r softwares-doris/realtime-new-doris/realtime-add/doris/jdbc_drivers  /data/doris/fe/
cp -r softwares-doris/realtime-new-doris/realtime-add/doris/jdbc_drivers  /data/doris/be/
```



### 4.3 BE节点部署

#### 4.3.1 安装目录分发
`将上面在FE管理节点修改完毕后的doris安装目录分发至BE节点上`  
```shell
scp -r /data/doris/ realtime-dw1:/data/
scp -r /data/doris/ realtime-dw2:/data/
scp -r /data/doris/ realtime-dw3:/data/
......
```
#### 4.3.2 BE配置文件修改/data/doris/be/conf/be.conf（所有BE节点）
```shell
# 参数配置  【涉及网络和IP的参数请按实际环境修改】
echo "priority_networks = 10.0.0.6/24" >> /data/doris/be/conf/be.conf # 修改为当前be节点的ip，以doris-test2为例
echo "JAVA_HOME=/usr/local/jdk/" >> /data/doris/be/conf/be.conf            # 配置jdk家目录 
```

## 5.doris集群启动
### 5.1 启动FE（realtime-dw节点）
```shell
/data/doris/fe/bin/start_fe.sh --daemon
```

### 5.2 启动BE节点（realtime-dw1/2/3）
```shell
/data/doris/be/bin/start_be.sh --daemon
```

### 5.3 将BE节点加入到FE管理节点中 （realtime-dw节点）
#### 5.3.1 登录mysql并设置初始密码和最大连接数
```sql
mysql -hrealtime-dw -uroot -P 19030                # 默认root无密码
SET PASSWORD FOR 'root' = PASSWORD('zanalytics'); # 修改root密码为zanalytics
SET PROPERTY FOR 'root' 'max_user_connections' = '1000'; #修改root最大连接数为1000，默认100
SET PASSWORD FOR 'admin' = PASSWORD('zanalytics'); # 修改admin密码为zanalytics
```
#### 5.3.2 添加BE节点
```sql
mysql -h realtime-dw -P 19030 -uroot -p'zanalytics'  # 登录doris后，添加be节点信息
MySQL [(none)]> ALTER SYSTEM ADD BACKEND "realtime-dw1:19050";
MySQL [(none)]> ALTER SYSTEM ADD BACKEND "realtime-dw2:19050";
MySQL [(none)]> ALTER SYSTEM ADD BACKEND "realtime-dw3:19050";
```

#### 5.3.3 查看BE节点状态

```sql
MySQL [(none)]> show proc '/backends';
Alive为true表示该BE节点存活

```



## 6.doris数仓初始化

`文档中涉及到的sql文件，可在基础安装包中找到：softwares-doris/realtime-new-doris/realtime-add/doris`

### 6.1 库表初始化


```sql
####################################################################################################
`注意：非标注环境部署时，如果使用客户提供的mysql或者mysql使用的用户名、密码、端口非默认的（web\zanalytics\3306），
需要在初始化doris前将初始化sql中2450行 'CREATE CATALOG IF NOT EXISTS mysql_sdkv PROPERTIES'中
涉及的user、password、jdbc_url修改为实际使用的信息。
CREATE CATALOG IF NOT EXISTS mysql_sdkv PROPERTIES (
    "type" = "jdbc",
    "user" = "web",
    "password" = "zanalytics",
    "jdbc_url" = "jdbc:mysql://realtime-db:3306/sdkv?useUnicode=true&characterEncoding=utf8&autoReconnect=true",
    "driver_url" = "mysql-connector-java-8.0.23.jar",
    "driver_class" = "com.mysql.cj.jdbc.Driver"
);`
###################################################################################################


mysql -hrealtime-dw -uroot -P 19030 -p'zanalytics'  # 登录doris
MySQL [(none)]> CREATE DATABASE IF NOT EXISTS zanalytics;         # 创建zanalytics库
MySQL [(none)]> CREATE DATABASE IF NOT EXISTS ods;  # 创建ods库
MySQL [(none)]> use zanalytics; # 切换zanalytics库，执行初始化表操作
MySQL [(none)]> source /xxx/1_init_flush_tables_doris.sql   # 请输入实际sql文件路径
```

### 6.2 UDF函数注册（java自定义udf函数）

#### 6.2.1 函数创建
```sql
# 登录doris后，创建函数
mysql -hrealtime-dw -uroot -P 19030 -p'zanalytics'  
```
##### 6.2.2.1 页面路径功能——依赖函数 `instr`
```sql
CREATE GLOBAL FUNCTION instr(String,String,int,int) RETURNS int PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.InstrUDF",
    "always_nullable"="false",
    "type"="JAVA_UDF"
);

# 验证：
select instr('a1|,|a2|,|a3','|,|',1,2);  
# 返回： 8
```
##### 6.2.2.2 用户旅程功能——依赖函数 `pagePathMatch`
```sql
CREATE GLOBAL FUNCTION pagePathMatch(String,String) RETURNS String PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.PathMatchUDF",
    "always_nullable"="true",
    "type"="JAVA_UDF"
);
# 验证：
select pagePathMatch ('001md51:1,002md52:2,003md54:1,004md54:1','{"stage1":["md51","md52"],"stage2":["md53","md52"],"stage3":["md54","md52"]}');
# 返回：  md51:1:1-md52:2,md52:2:2-md54:1,md54:1:3
```
##### 6.2.2.3 系统相关操作-依赖函数 `splitPartIndex`
```sql
CREATE GLOBAL FUNCTION splitPartIndex(String,String,int,int) RETURNS String PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.splitPartIndexUDF",
    "always_nullable"="true",
    "type"="JAVA_UDF"
);
```
##### 6.2.2.4 系统相关操作-依赖函数 `idCardParse`
```sql
CREATE GLOBAL FUNCTION idCardParse(String,Integer,String) RETURNS String PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.IDCardParserUDF",
    "always_nullable"="true",
    "type"="JAVA_UDF"
);
```
##### 6.2.2.5 系统相关操作-依赖函数 `zg_yearweek`
```sql
CREATE GLOBAL FUNCTION zg_yearweek(String) RETURNS int PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.WeekYearUDF",
    "always_nullable"="true",
    "type"="JAVA_UDF"
);
CREATE GLOBAL FUNCTION zg_yearweek(String,int) RETURNS int PROPERTIES (
    "file"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
    "symbol"="io.zhuge.etl.impalaUDF.WeekYearUDF",
    "always_nullable"="true",
    "type"="JAVA_UDF"
);
```
##### 6.2.2.6 查看doris集群目前已创建udf函数
```sql
MySQL [(none)]> SHOW GLOBAL FUNCTIONS;
+----------------+
| Function Name  |
+----------------+
| idcardparse    |
| instr          |
| pagepathmatch  |
| splitpartindex |
| zg_yearweek    |
+----------------+
5 rows in set (0.00 sec)
```

##### 6.2.2.7 创建中位数函数【非全局函数，需要切到zanalytics库执行】

```sql
MySQL [(none)]> use zanalytics;
MySQL [zanalytics]> CREATE AGGREGATE FUNCTION appx_median(DOUBLE) RETURNS DOUBLE PROPERTIES (
  "SYMBOL"="io.zhuge.etl.impalaUDF.MedianUDAF",
  "FILE"="file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar",
  "ALWAYS_NULLABLE"="true",
  "TYPE"="JAVA_UDF"
);

#验证：
MySQL [zanalytics]> show full functions;
+---------------------+-------------+---------------+-------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| Signature           | Return Type | Function Type | Intermediate Type | Properties                                                                                                                                                                          |
+---------------------+-------------+---------------+-------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
| appx_median(DOUBLE) | DOUBLE      | Aggregate     | NULL              | {"symbol":"io.zhuge.etl.impalaUDF.MedianUDAF","object_file":"file:///data/doris/udf/etl-impalaUDF-2.0-SNAPSHOT-jar-with-dependencies.jar","md5":"60f55f3fc8170af33e4fd9f464c882f7"} |
+---------------------+-------------+---------------+-------------------+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
1 row in set (0.00 sec)
【注意：后续如有jar包更新，函数返回的MD5以实际使用的jar包为准】

```

##### 6.2.8 fe和be自动拉起

将以下脚本放到/data/doris/ ,然后添加计划任务5分钟执行一次

```bash
# /data/doris/fe_status.sh
# FE自动拉起脚本：
#!/bin/sh
source /etc/profile
now=$(date +`%Y-%m-%d %H:%M%S`)
echo "$now, doris_fe monitor ..."

ps -ef |grep org.apache.doris.DorisFE |grep -v grep
if [ $? -ne 0 ]
then 
  echo "doris fe is bad ......"
  echo "restarting doris_fe"
  sh /data/doris/fe/bin/start_fe.sh  --daemon
else
  echo "doris fe is running ..... "
fi
```

```bash
# /data/doris/be_status.sh
# BE自动拉起脚本
#!/bin/sh
source /etc/profile
now=$(date +`%Y-%m-%d %H:%M%S`)
echo "$now, doris_be monitor ..."

ps -ef |grep 'lib/doris_be' |grep -v grep
if [ $? -ne 0 ]
then 
  echo "doris be is bad ......"
  echo "restarting doris_be"
  sh /data/doris/be/bin/start_be.sh  --daemon
else
  echo "doris fe is running ..... "
fi
```



### 6.3 fe高可用（按需部署）

#### 6.3.1 fe多节点部署

```
FE角色：
Follower——master + follower ，3节点组成高可用，一个推荐3节点【奇数】
Observer——不参与高可用选举，用于扩展FE的读服务能力

FE多节点部署
1）当前给客户推荐的一般是至少3台服务器，建议是5台以上
2）当前doris集群大部分是FE、BE混部，【强烈要求】元数据目录和数据目录分属不同磁盘
3）FE多节点部署主要步骤
fe.conf 配置文件：
priority_networks = xx.xx.xx.xx/24 #注意不同节点IP配置不同
```

```
第1个节点启动：
[root@realtime-dw1 ~]# /data/doris/fe/bin/start_fe.sh --daemon
第2、3个节点启动：
[root@realtime-dw2 ~]# /data/doris/fe/bin/start_fe.sh  --helper realtime-dw1:19010 --daemon
[root@realtime-dw3 ~]# /data/doris/fe/bin/start_fe.sh  --helper realtime-dw1:19010 --daemon
注意： --helper 参数仅在 follower 和 observer 第一次启动时才需要。
```

```
登录doris，注册fe节点：
mysql -h realtime-dw -P 19030 -uroot -p'zanalytics'  
MySQL [(none)]> ALTER SYSTEM ADD FOLLOWER "<realtime-dw2_IP>:19010";  #注意：这里使用主机名会连接失败一定要使用IP
MySQL [(none)]> ALTER SYSTEM ADD FOLLOWER "<realtime-dw3_IP>:19010";  #注意：这里使用主机名会连接失败一定要使用IP
MySQL [(none)]> show frontends;  

```

登录WEBUI查看fe状态：

![fe-ha_webui](https://gl.zhugeio.com/yunwei/private_cluster_doris/blob/master/Docs/fe-ha_webui.png)

#### 6.3.2 fe多节点高可用



参考：https://doris.apache.org/zh-CN/docs/admin-manual/cluster-management/load-balancing
以haproxy实现方式为例，使用最新的docker.io/library/haproxy:3.2-alpine镜像容器化部署：

提前编写haproxy.cfg配置文件放到 /data/haproxy/etc/：

```
global
    maxconn         100000
    ulimit-n        65536
    nbthread		2
    #log             stdout  format raw  local0  info
    log             /dev/log local0 warning
    #uid             99
    #gid             99
    #chroot          /var/lib/haproxy/
    daemon
    stats	    	 socket 0.0.0.0:19998 mode 664 level admin
    stats	    	 timeout  60s
    #user            haproxy
    #group           haproxy

defaults
    log     global
    mode    http
    retries 3
    option  redispatch
    option  abortonclose
    timeout connect 5000
    timeout client  60000
    timeout server  60000
    timeout check   2000

userlist stats-auth
    group admin    users admin
    user  admin    insecure-password Zgio@123
    group readonly users zgio
    user  zgio     insecure-password haproxy

listen stats:
    bind  *:19999
    mode  http
    stats enable
    log   global
    stats hide-version
    stats uri /status
    stats refresh 10s
    acl AUTH       http_auth(stats-auth)
    acl AUTH_ADMIN http_auth_group(stats-auth) admin
    stats http-request auth unless AUTH
    stats admin if AUTH_ADMIN

frontend front-agent
    bind *:19030
    mode tcp
    default_backend fe-forward

backend fe-forward
    mode    tcp
    balance leastconn
    server  fe-1  realtime-dw1:19030 weight 1 check inter 3000 rise 2 fall 3
    server  fe-2  realtime-dw2:19030 weight 1 check inter 3000 rise 2 fall 3
    server  fe-3  realtime-dw3:19030 weight 1 check inter 3000 rise 2 fall 3

frontend front-http
    bind *:18030
    mode http
    default_backend fe-http

backend fe-http
    mode    http
    balance first
    server  fe-1  realtime-dw1:18030 weight 1 check inter 3000 rise 2 fall 3
    server  fe-2  realtime-dw2:18030 weight 1 check inter 3000 rise 2 fall 3
    server  fe-3  realtime-dw3:18030 weight 1 check inter 3000 rise 2 fall 3
    
```

```bash
 拉取镜像：
 docker pull haproxy:3.2-alpine
 离线环境下镜像文件可在部署包 【softwares-doris/realtime-new-doris/realtime/web_all/realtime-images/ 】路径下获取，将镜像复制到本机然后直接导入：
 docker load < haproxy3.2-alpine.tar
 启动容器：
 podman run -d --name haproxy32 \
         -v /data/haproxy/etc:/usr/local/etc/haproxy \
         -v /etc/hosts:/etc/hosts -v /dev/log:/dev/log \
         --restart=unless-stopped \
         -p 19999:19999 -p 19998:19998 -p 19030:19030 -p 18030:18030 \
         --sysctl net.ipv4.ip_unprivileged_port_start=0  haproxy:3.2-alpine
```


mysql访问haproxy：

![fe-ha__mysql](https://gl.zhugeio.com/yunwei/private_cluster_doris/blob/master/Docs/fe-ha_mysql.png)

访问haproxy机器的IP/status查看haproxy状态：

![fe-ha_status](https://gl.zhugeio.com/yunwei/private_cluster_doris/blob/master/Docs/fe-ha_status.png)

提前安装socat用于haproxy的socket通信，查看后端server状态：

![fe-ha_socat](https://gl.zhugeio.com/yunwei/private_cluster_doris/blob/master/Docs/fe-ha_socat.png)

后续可在不重启haproxy的情况下通过socat对后端server动态执行add\del\set\enable\disable等维护操作，
详细操作可参考官方文档：https://www.haproxy.com/documentation/haproxy-runtime-api/reference/set-server/



```
fe高可用的目的是为了防止单节点fe在元数据损坏的情况下造成集群不可用。
实际交付部署时，可在不改变部署流程和节点分布的情况下，
通过注册Observer的方式实现fe的元数据meta-data副本备份。
此种方式在fe主节点元数据损坏时需要停服修复元数据。
```



## 7.其他（了解即可）

### 7.1 Doris集群启停
```plain
# doris 集群启停
# 关闭
  /data/doris/be/bin/stop_be.sh
  /data/doris/fe/bin/stop_fe.sh
# 启动
  /data/doris/fe/bin/start_fe.sh --daemon
  /data/doris/be/bin/start_be.sh --daemon
Ps: FE在realtime-dw节点，BE在realtime-dw1/2/3节点
```

### 7.2 Doris集群端口简介
```
网络需求:Doris 各个实例直接通过网络进行通讯。以下表格展示了所有需要的端口  
```
| 实例名称 | 端口名称 | 默认端口 | 实际使用端口 | 通讯方向 | 说明 |
| --- | --- | --- | --- | --- | --- |
| BE | be_port | 9060 | 19060 | FE --> BE | BE 上 thrift server 的端口，用于接收来自 FE 的请求 |
| BE | webserver_port | 8040 | 18040 | BE <--> BE | BE 上的 http server 的端口 |
| BE | heartbeat_service_port | 9050 | 19050 | FE --> BE | BE 上心跳服务端口（thrift），用于接收来自 FE 的心跳 |
| BE | brpc_port | 8060 | 18060 | FE <--> BE, BE <--> BE | BE 上的 brpc 端口，用于 BE 之间通讯 |
| FE | http_port | 8030 | 18030 | FE <--> FE，用户 <--> FE | FE 上的 http server 端口 |
| FE | rpc_port | 9020 | 19020 | BE --> FE, FE <--> FE | FE 上的 thrift server 端口，每个fe的配置需要保持一致 |
| FE | query_port | 9030 | 19030 | 用户 <--> FE | FE 上的 mysql server 端口 |
| FE | edit_log_port | 9010 | 19010 | FE <--> FE | FE 上的 bdbje 之间通信用的端口 |
| Broker | broker_ipc_port | 8000 | 18000 | FE --> Broker, BE --> Broker | Broker 上的 thrift server，用于接收请求 |


### 7.3 Doris集群web访问地址
#### 7.3.1 FE的web访问地址
```plain
http://${FE主机IP}:18030
账号密码：root/zanalytics
```
#### 7.3.2 BE的web访问地址
```plain
http://${FE主机IP}:18040
```

#### 7.3.3 节点缩容：

##### be节点下线缩容可以选择 DROP 或 DECOMMISSION 两种方案：

|                  | DROP                           | DECOMMISSION                                                 |
| ---------------- | ------------------------------ | ------------------------------------------------------------ |
| 下线原理         | 直接下线节点，删除掉 BE 节点。 | 发起命令后，会尝试将该 BE 数据迁移到其他节点上，当迁移完成后，BE 节点自动下线。 |
| 生效周期         | 执行后立即生效。               | 待数据搬迁完成后，删除命令生效。根据集群现有数据量，可能在小时到 1 天不等时间内。 |
| 副本表处理方案   | 可能会造成数据丢失             | 不会造成数据丢失。                                           |
| 同时下线多个节点 | 可能会造成数据丢失。           | 不会造成数据丢失。                                           |
| 生产推荐         | 不建议生产环境使用。           | 推荐在生产环境使用。                                         |

	DECOMMISSION 命令说明：
	
	DECOMMISSION 是一个异步操作。执行后，可以通过 SHOW backends; 看到该 BE 节点的 SystemDecommissioned 状态为 true。表示该节点正在进行下线；
	
	DECOMMISSION 命令可能会执行失败，如剩余 BE 存储空间不足以容纳下线 BE 上的数据，或者剩余机器数量不满足最小副本数时，该命令都无法完成，并且 BE 会一直处于 SystemDecommissioned 为 true 的状态；
	
	DECOMMISSION 的进度，可以通过 SHOW PROC '/backends'; 中的 TabletNum 查看，如果正在进行，TabletNum 将不断减少；
	
	可以通过 CANCEL DECOMMISSION BACKEND "be_host:be_heartbeat_service_port"; 命令取消。取消后，该 BE 上的数据将维持当前剩余的数据量。后续 Doris 重新进行负载均衡；
	
	可以调整 balance_slot_num_per_path 参数调整数据搬迁速率。

通过以下命令，可以使用 DROP 方式删除 BE 节点：

```sql
ALTER SYSTEM DROP backend "realtime-dw4:19050";
```

通过以下命令，可以使用 DECOMMISSION 方式删除 BE 节点：

```sql
ALTER SYSTEM DECOMMISSION backend "realtime-dw4:19050";
```

##### 在缩容 FE 节点时，也要保证最终集群内 Master 与 Follower 节点总和为奇数个，通过以下命令可以缩容节点：

```sql
ALTER SYSTEM DROP FOLLOWER "realtime-dw5:19010";
ALTER SYSTEM DROP OBSERVER "realtime-dw6:19010";
```

