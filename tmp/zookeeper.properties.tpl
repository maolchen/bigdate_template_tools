# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
# 
#    http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# the directory where the snapshot is stored.
dataDir={{config.zookeeper.dataDir}}
# the port at which the clients will connect
clientPort={{config.zookeeper.clientPort}}
# disable the per-ip limit on the number of connections since this is a non-production config
maxClientCnxns=0
initLimit=5
syncLimit=2

{% for host in groups['etl_kafka'] %}
server.{{ loop.index }}={{ host }}:2889:3889
{% endfor %}
#server.1=realtime-dw1:2889:3889
#server.2=realtime-dw2:2889:3889
#server.3=realtime-dw3:2889:3889
