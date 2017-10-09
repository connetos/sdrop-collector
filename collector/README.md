# Collector

This sdrop collector is based grafana, use grafana as the user interface to present the information.

- directory agent contain the agent of sdrop, which parse the packet and add it to database.
- directory ui contain the dashboard configuration of grafana.

grafana is a powerful frontend to represent metric, it supports all kinds of datasource, we use mysql to hold all the information.

Below is UI of the sdrop collector.

<img src="./sdrop.png" width = "900" height = "600" alt="sdrop grafana" align=center />


## Install step

Follow the step you will set up the sdrop collector.

### Install Grafana
Grafana is published on [Github](https://github.com/grafana/grafana), go get it's code, then build it.

#### Get grafana code
<pre>
go get github.com/grafana/grafana
</pre>

if you want to build with a forked repo, do as the following
<pre>
go get github.com/xxxx/grafana
mkdir $GOPATH/src/github.com/grafana
ln -s  $GOPATH/src/github.com/xxxx/grafana $GOPATH/src/github.com/grafana/grafana
</pre>

#### Build backend
<pre>
cd $GOPATH/src/github.com/grafana/grafana
go run build.go setup
go run build.go build
</pre>

#### Build frontend    
<pre>
npm install -g yarn
yarn install --pure-lockfile
npm run build
</pre>

#### Running grafana
<pre>
cd bin
./granfana-server
</pre>    

Lauch the web with URL, default username password admin/admin.<br>
[http://xxxxx:3000/](http://xxxxx:3000/)

#### Build And Running agent

<pre>
go get github.com/connetos/sdrop_collector
cd $GOPATH/src/github.com/connetos/sdrop_collector/collector/agent
go build
./agent
</pre>

The Agent assume the device uses the UDP 32768 as the port the export dropped packets, if you changed to other then you should modify the code below.
<pre>
const (
    MyDB = "drop_packets"
    service = ":32768"
    username = "root"
    password = "mysqlpass"
    DbUrl = username + ":" + password + "@/" + MyDB + "?charset=utf8"
    hexDigit = "0123456789abcdef"
)
</pre>

### Install Mysql

Install Mysql by package manage tools like apt-get/yum etc.<br>
Set the Mysql database password, will be used in the agent.<br>

<pre>
const (
    MyDB = "drop_packets"
    service = ":32768"
    username = "root"
    password = "mysqlpass"
    DbUrl = username + ":" + password + "@/" + MyDB + "?charset=utf8"
    hexDigit = "0123456789abcdef"
)
</pre>

### Import database
Lauch the Mysql CLI, source the database.sql.

<pre>
mysql -uroot -p
Enter password:
Welcome to the MySQL monitor.  Commands end with ; or \g.
Your MySQL connection id is 51459
Server version: 5.5.57-0+deb8u1 (Debian)

Copyright (c) 2000, 2017, Oracle and/or its affiliates. All rights reserved.

Oracle is a registered trademark of Oracle Corporation and/or its
affiliates. Other names may be trademarks of their respective
owners.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

mysql> source path/to/database.sql
mysql> use drop_packets
Reading table information for completion of table and column names
You can turn off this feature to get a quicker startup with -A

Database changed
mysql> show tables;
+------------------------------+
| Tables_in_drop_packets       |
+------------------------------+
| drop_packet_arp_info         |
| drop_packet_ip_protocol_info |
| drop_packet_ipv4_info        |
| drop_packet_layer2_info      |
| drop_packet_meta_info        |
| drop_packet_vlan_tag_info    |
+------------------------------+
6 rows in set (0.00 sec)
</pre>

### Import Grafana dashboard

Please refer to README.md in ui directory.