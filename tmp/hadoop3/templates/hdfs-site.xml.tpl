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
  <name>dfs.permissions.superusergroup</name>
  <value>hadoop</value>
 </property>

 <property>
  <name>fs.permissions.umask-mode</name>
  <value>022</value>
 </property>

 <property>
  <name>dfs.namenode.acls.enabled</name>
  <value>true</value>
 </property>

 <property>
  <name>dfs.datanode.data.dir</name>
  <value>file:///data/dfs/dn</value>
 </property>

 <property>
  <name>dfs.blocksize</name>
  <value>268435456</value>
 </property>
 
 <property>
  <name>dfs.nameservices</name>
  <value>zhugeio</value>
 </property>
 
 <property>
  <name>dfs.namenode.name.dir</name>
  <value>file:///data/dfs/nn</value>
 </property>
 
 <property>
  <name>dfs.ha.namenodes.zhugeio</name>
  <value>realtime-dw1,realtime-dw2</value>
 </property>
 
 <property>
  <name>dfs.namenode.rpc-address.zhugeio.realtime-dw1</name>
  <value>realtime-dw1:8020</value>
 </property>
 
 <property>
  <name>dfs.namenode.rpc-address.zhugeio.realtime-dw2</name>
  <value>realtime-dw2:8020</value>
 </property>

 <property>
  <name>dfs.namenode.http-address.zhugeio.realtime-dw1</name>
  <value>realtime-dw1:50070</value>
 </property>

 <property>
  <name>dfs.namenode.http-address.zhugeio.realtime-dw2</name>
  <value>realtime-dw2:50070</value>
 </property>

 <property>
  <name>dfs.namenode.shared.edits.dir</name>
  <value>qjournal://realtime-dw1:8485;realtime-dw2:8485;realtime-dw3:8485/zhugeio</value>
 </property>

 <property>
  <name>dfs.client.failover.proxy.provider.zhugeio</name>
  <value>org.apache.hadoop.hdfs.server.namenode.ha.ConfiguredFailoverProxyProvider</value>
 </property>
 
 <property>
  <name>dfs.ha.fencing.ssh.private-key-files</name>
  <value>/root/.ssh/id_dsa</value>
 </property>

 <property>
  <name>dfs.ha.fencing.methods</name>
  <value>shell(/bin/true)</value>
 </property>

 <property>
  <name>dfs.journalnode.edits.dir</name>
  <value>/data/dfs/jn</value>
 </property>

 <property>
  <name>dfs.ha.automatic-failover.enabled</name>
  <value>true</value>
 </property>

 <property>
  <name>dfs.datanode.hdfs-blocks-metadata.enabled</name>
  <value>true</value>
 </property>

 <property>
  <name>dfs.client.read.shortcircuit</name>
  <value>true</value>
 </property>

 <property>
  <name>dfs.domain.socket.path</name>
  <value>/data/hadoop3/hdfs-sockets/dn</value>
 </property>

</configuration>
