create database drop_packets;

use drop_packets;

/*
drop_packet_meta_info
+-------+-----------+----------+----------+-----------+---------------+-------------------+--------------------+-----------------------+
|  id   |  device   | ingress  |  egress  |   reason  |   timestamp   |   reason string   |   sample packets   |  layer2/tag/3/4 index |
+-------+-----------+----------+----------+-----------+---------------+-------------------+--------------------+-----------------------+
|       |           |          |          |           |               |                   |                    |                       |
+-------+-----------+----------+----------+-----------+---------------+-------------------+--------------------+-----------------------+
*/

CREATE TABLE drop_packet_meta_info( 
    id INT NOT NULL AUTO_INCREMENT,
    device VARCHAR(32) NOT NULL,
    ingress_interface VARCHAR(32) NOT NULL,
    egress_interface VARCHAR(32) NOT NULL, 
    drop_reason INT NOT NULL, 
    drop_reason_string VARCHAR(100) NOT NULL, 
    last_detected_time TIMESTAMP, 
    sample_packets INT NOT NULL, 
    layer2_index INT NOT NULL, 
    tag_index INT NOT NULL, 
    arp_index INT NOT NULL, 
    layer3_index INT NOT NULL, 
    layer4_index INT NOT NULL, 
    PRIMARY KEY ( id )) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*
drop_packet_layer2_info
+---------+-----------+----------+------------------+---------------+-----------------+
|  id     |   dmac    |   smac   |    ether_type    |     length    |    timestamp    |
+---------+-----------+----------+------------------+---------------+-----------------+
|         |           |          |                  |               |                 |
+---------+-----------+----------+------------------+---------------+-----------------+
*/

CREATE TABLE drop_packet_layer2_info(
    id INT NOT NULL AUTO_INCREMENT,
    dmac VARCHAR(64) NOT NULL,
    smac VARCHAR(64) NOT NULL,
    ether_type VARCHAR(32) NOT NULL,
    length INT NOT NULL,
    last_detected_time TIMESTAMP, 
    PRIMARY KEY ( id ))ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*
INSERT INTO drop_packet_vlan_tag_info (dmac, smac, ether_type, length, vlan_tag_index, layer3_index)
VALUES
("00:00:00:11:22:33", "00:00:11:33:22:44", 800, 128, 0, 0)
*/

/*
drop_packet_vlan_tag_info
+---------+-----------+----------+------------+
|  id     |   tpid    |  vlanid  |  timestamp |
+---------+-----------+----------+------------+
|         |           |          |            |
+---------+-----------+----------+------------+
*/

CREATE TABLE drop_packet_vlan_tag_info(
    id INT NOT NULL AUTO_INCREMENT,
    tpid VARCHAR(32) NOT NULL,
    vlanid SMALLINT NOT NULL,
    last_detected_time TIMESTAMP, 
    PRIMARY KEY ( id ))ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*
drop_packet_arp_info
+---------+-----------+----------+----------+----------+----------+------------+
|  id     |   opcode  |  snd mac |  snd ip  |   t mac  |   t ip   |  timestamp |
+---------+-----------+----------+----------+----------+----------+------------+
|         |           |          |          |          |          |            |
+---------+-----------+----------+----------+----------+----------+------------+
*/

CREATE TABLE drop_packet_arp_info( 
    id INT NOT NULL AUTO_INCREMENT, 
    opcode TINYINT NOT NULL, 
    sender_mac VARCHAR(64) NOT NULL,
    target_mac VARCHAR(64) NOT NULL,
    sender_ip VARCHAR(64) NOT NULL, 
    target_ip VARCHAR(64) NOT NULL, 
    last_detected_time TIMESTAMP, 
    PRIMARY KEY ( id ))ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*
drop_packet_ipv4_info
+---------+-----------+----------+----------+----------+----------+----------+------------+
|  id     |    sip    |    dip   |    tos   |   len    |   ttl    |   proto  |  timestamp |
+---------+-----------+----------+----------+----------+----------+----------+------------+
|         |           |          |          |          |          |          |            |
+---------+-----------+----------+----------+----------+----------+----------+------------+
*/

CREATE TABLE drop_packet_ipv4_info( 
    id INT NOT NULL AUTO_INCREMENT, 
    source_ip VARCHAR(64) NOT NULL, 
    destination_ip VARCHAR(64) NOT NULL, 
    tos INT NOT NULL, 
    length INT NOT NULL,
    ttl INT NOT NULL, 
    protocol VARCHAR(64) NOT NULL, 
    last_detected_time TIMESTAMP, 
    PRIMARY KEY ( id ))ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*
drop_packet_ip_protocol_info
+---------+-----------+----------+------------+ 
|  id     |  l4 sport | l4 dport |  timestamp | 
+---------+-----------+----------+------------+ 
|         |           |          |            | 
+---------+-----------+----------+------------+ 
*/

CREATE TABLE drop_packet_ip_protocol_info( 
    id INT NOT NULL AUTO_INCREMENT, 
    l4_source_port VARCHAR(64) NOT NULL, 
    l4_destination_port VARCHAR(64) NOT NULL, 
    last_detected_time TIMESTAMP, 
    PRIMARY KEY ( id ))ENGINE=InnoDB DEFAULT CHARSET=utf8; 

