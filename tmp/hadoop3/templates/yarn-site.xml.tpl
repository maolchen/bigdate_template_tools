<?xml version="1.0"?>
<!--
  Licensed to the Apache Software Foundation (ASF) under one or more
  contributor license agreements.  See the NOTICE file distributed with
  this work for additional information regarding copyright ownership.
  The ASF licenses this file to You under the Apache License, Version 2.0
  (the "License"); you may not use this file except in compliance with
  the License.  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
-->
<?xml-stylesheet type="text/xsl" href="configuration.xsl"?>

<configuration>

  <property>
    <name>yarn.nodemanager.aux-services</name>
    <value>mapreduce_shuffle</value>
  </property>

  <property>
    <name>yarn.nodemanager.aux-services.mapreduce_shuffle.class</name>
    <value>org.apache.hadoop.mapred.ShuffleHandler</value>
  </property>

 <!--  <property>
    <name>yarn.log-aggregation-enable</name>
    <value>true</value>
  </property> -->

  <property>
    <description>List of directories to store localized files in.</description>
    <name>yarn.nodemanager.local-dirs</name>
    <value>file:///data/hadoop3/hadoop-yarn/cache/${user.name}/nm-local-dir</value>
  </property>

  <property>
    <description>Where to store container logs.</description>
    <name>yarn.nodemanager.log-dirs</name>
    <value>file:///data/hadoop3/hadoop-yarn/containers</value>
  </property>

  <property>
    <description>Classpath for typical applications.</description>
     <name>yarn.application.classpath</name>
     <value>
        $HADOOP_CONF_DIR,
        $HADOOP_COMMON_HOME/*,$HADOOP_COMMON_HOME/lib/*,
        $HADOOP_HDFS_HOME/*,$HADOOP_HDFS_HOME/lib/*,
        $HADOOP_MAPRED_HOME/*,$HADOOP_MAPRED_HOME/lib/*,
        $HADOOP_YARN_HOME/*,$HADOOP_YARN_HOME/lib/*
     </value>
  </property>

 <property>
   <name>yarn.resourcemanager.ha.enabled</name>
   <value>true</value>
 </property>

 <property>
   <name>yarn.resourcemanager.cluster-id</name>
   <value>zhugeio2</value>
 </property>

 <property>
   <name>yarn.resourcemanager.ha.rm-ids</name>
   <value>rm1,rm2</value>
 </property>

 <property>
   <name>yarn.resourcemanager.hostname.rm1</name>
   <value>realtime-dw1</value>
 </property>

 <property>
   <name>yarn.resourcemanager.hostname.rm2</name>
   <value>realtime-dw2</value>
 </property>

 <property>
   <name>yarn.resourcemanager.zk-address</name>
   <value>{% for host in groups['etl_kafka'] %}{{ host }}:2182{% if not loop.last %},{% endif %}{% endfor %}</value>
 </property>

 <property>  
   <name>yarn.resourcemanager.webapp.address.rm1</name>  
   <value>realtime-dw1:8089</value>  
 </property> 

 <property>  
   <name>yarn.resourcemanager.webapp.address.rm2</name>  
   <value>realtime-dw2:8089</value>  
 </property>

 <property>
   <name>yarn.nodemanager.vmem-pmem-ratio</name>
   <value>5</value>
 </property>

 <property>
   <name>yarn.scheduler.minimum-allocation-mb</name>
   <value>32</value>
 </property>

 <property>  
   <name>yarn.nodemanager.remote-app-log-dir</name>
   <value>hdfs://zhugeio/var/log/hadoop-yarn/apps</value>
 </property>

 <property>
   <name>yarn.resourcemanager.scheduler.class</name>
   <value>org.apache.hadoop.yarn.server.resourcemanager.scheduler.capacity.CapacityScheduler</value>
 </property>

 <property>  
   <name>yarn.scheduler.maximum-allocation-vcores</name>  
   <value>8</value>
 </property>

 <property>
  <name>yarn.nodemanager.delete.debug-delay-sec</name>
  <value>86400</value>
 </property>

 <property>
  <name>yarn.nodemanager.log.retain-seconds</name>
  <value>86400</value>
 </property>

 <property>
   <name>yarn.nodemanager.resource.memory-mb</name>
   <value>12288</value>
   <description>Amount of physical memory, in MB, that can be allocated for containers.</description>
 </property>

</configuration>
