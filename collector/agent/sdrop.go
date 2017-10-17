package main

import (
    "fmt"
    "net"
    "os"
    "time"
    "strings"
    "regexp"
    "strconv"
    "encoding/hex"

    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"

    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

const (
    MyDB = "drop_packets"
    service = ":32768"
    username = "root"
    password = "mysqlpass"
    DbUrl = username + ":" + password + "@/" + MyDB + "?charset=utf8"
    hexDigit = "0123456789abcdef"
)

// Convert integer to decimal string.
func itoa(val int) string {
	if val < 0 {
		return "-" + uitoa(uint(-val))
	}
	return uitoa(uint(val))
}

// Convert unsigned integer to decimal string.
func uitoa(val uint) string {
	if val == 0 { // avoid string allocation
		return "0"
	}
	var buf [20]byte // big enough for 64bit value base 10
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	// val < 10
	buf[i] = byte('0' + val)
	return string(buf[i:])
}

func IPToString(ip []byte) string {
	if len(ip) != 4 {
		return "<nil>"
	}

    return uitoa(uint(ip[0])) + "." +
        uitoa(uint(ip[1])) + "." +
        uitoa(uint(ip[2])) + "." +
        uitoa(uint(ip[3]))
}

func macToString(a []byte) string {
	if len(a) == 0 {
		return ""
	}
	buf := make([]byte, 0, len(a)*3-1)
	for i, b := range a {
		if i > 0 {
			buf = append(buf, ':')
		}
		buf = append(buf, hexDigit[b>>4])
		buf = append(buf, hexDigit[b&0xF])
	}
	return string(buf)
}

func getElement(str string, key string) string {
    c := []byte(str)
    prefix := "<" + key + ">"
    suffix := "</" + key + ">"
    pat := prefix + "(.)*" + suffix
    reg := regexp.MustCompilePOSIX(pat)
    sub := reg.Find(c)
    initial := string(sub)
    r := strings.NewReplacer(prefix, "", suffix, "")
    out := r.Replace(initial)

    return out
}

func decodePacketData(packet string) []byte {
    data := []byte (packet)
    buf := make([]byte, hex.DecodedLen(len(data)))
    _, err := hex.Decode(buf, data)
    if err != nil {
        return nil
    }

    return buf
}

func updateDropPacketMeta(db *sql.DB,
    device string, ingress_interface string,
    egress_interface string, drop_reason int64, drop_reason_string string,
    timestamp string) int {

    // if old drop packet meta exists
    stmt, err := db.Prepare("SELECT id,sample_packets FROM drop_packet_meta_info " +
        "WHERE device=? AND ingress_interface=? AND egress_interface=? AND drop_reason=?")
    checkErr(err)

    _, err = stmt.Exec(device, ingress_interface, egress_interface, drop_reason)
    checkErr(err)

    var id int
    var sample_packets int
    err = stmt.QueryRow(device, ingress_interface, egress_interface, drop_reason).Scan(&id, &sample_packets)
    if (err == nil) {
        /* update exist one */
        stmt, err := db.Prepare("UPDATE drop_packet_meta_info " +
            "SET sample_packets=?,last_detected_time=? WHERE id=?")
        checkErr(err)

        _, err = stmt.Exec(sample_packets + 1, timestamp, id)
        checkErr(err)
    } else {
        /* insert new one */
        stmt, err := db.Prepare(
            "INSERT drop_packet_meta_info SET " +
            "device=?,ingress_interface=?,egress_interface=?,drop_reason=?," +
            "drop_reason_string=?,last_detected_time=?,sample_packets=?")
        checkErr(err)

        res, err := stmt.Exec(device, ingress_interface, egress_interface,
            drop_reason, drop_reason_string, timestamp, 1)
        checkErr(err)

        tmp, err := res.LastInsertId()
        checkErr(err)

        id = int(tmp)
    }

    stmt.Close();

    return id
}

func updateDropPacketEthernetInfo(db *sql.DB, packet gopacket.Packet,
    packet_length uint64, timestamp string, drop_index int) (string, string, string) {
    var id int
    var smac string
    var dmac string
    var etype string

    // Get the ethernet layer from this packet
    if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
        // Get actual ethernet data from this layer
        eth, _ := ethLayer.(*layers.Ethernet)
        smac = eth.SrcMAC.String()
        dmac = eth.DstMAC.String()
        etype = eth.EthernetType.String()
        length := packet_length

        stmt, err := db.Prepare("SELECT id FROM drop_packet_layer2_info " +
            "WHERE smac=? AND dmac=? AND ether_type=?")
        checkErr(err)

        _, err = stmt.Exec(smac, dmac, etype)
        checkErr(err)

        err = stmt.QueryRow(smac, dmac, etype).Scan(&id)
        if (err == nil) {
            /* update exist one */
            stmt, err := db.Prepare("UPDATE drop_packet_layer2_info " +
                "SET length=?,last_detected_time=?,drop_id=? where id=?")
            checkErr(err)

            _, err = stmt.Exec(length, timestamp, drop_index, id)
            checkErr(err)
        } else {
            /* insert new one */
            stmt, err := db.Prepare("INSERT drop_packet_layer2_info SET " +
                "smac=?,dmac=?,ether_type=?,length=?,last_detected_time=?,drop_id=?")
            checkErr(err)

            _, err = stmt.Exec(smac, dmac, etype, length, timestamp, drop_index)
            checkErr(err)
        }

        stmt.Close();
    }

    return dmac, smac, etype
}

func updateDropPacketVlanInfo(db *sql.DB, packet gopacket.Packet, ethertype string,
    vid uint16, timestamp string, drop_index int) (string, uint16) {

    var id int
    var vlanid uint16
    var etype string

    // Get the dot1q layer from this packet
    if dot1qLayer := packet.Layer(layers.LayerTypeDot1Q); dot1qLayer != nil {
        // Get actual dot1q data from this layer
        tag, _ := dot1qLayer.(*layers.Dot1Q)
        vlanid = tag.VLANIdentifier
        etype = tag.Type.String()

    } else {
        vlanid = vid
        etype = ethertype
    }

    stmt, err := db.Prepare("SELECT id FROM drop_packet_vlan_tag_info " +
    "WHERE tpid=? AND vlanid=?")
    checkErr(err)

    _, err = stmt.Exec(etype, vlanid)
    checkErr(err)

    err = stmt.QueryRow(etype, vlanid).Scan(&id)
    if (err == nil) {
        /* update exist one */
        stmt, err := db.Prepare("UPDATE drop_packet_vlan_tag_info " +
        "SET last_detected_time=?,drop_id=? WHERE id=?")
        checkErr(err)

        _, err = stmt.Exec(timestamp, drop_index, id)
        checkErr(err)
    } else {
        /* insert new one */
        stmt, err := db.Prepare("INSERT drop_packet_vlan_tag_info SET " +
        "tpid=?,vlanid=?,last_detected_time=?,drop_id=?")
        checkErr(err)

        _, err = stmt.Exec(etype, vlanid, timestamp, drop_index)
        checkErr(err)
    }

    stmt.Close()

    return etype, vlanid
}

func updateDropPacketArpInfo(db *sql.DB, packet gopacket.Packet,
    timestamp string, drop_index int) (uint16, string, string, string, string) {
    var id int
    var op uint16
    var sip string
    var smac string
    var tip string
    var tmac string

    // Get the arp layer from this packet
    if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
        // Get actual arp data from this layer
        arp, _ := arpLayer.(*layers.ARP)
        sip = IPToString(arp.SourceProtAddress)
        smac = macToString(arp.SourceHwAddress)
        tip = IPToString(arp.DstProtAddress)
        tmac = macToString(arp.DstHwAddress)
        op = arp.Operation

        stmt, err := db.Prepare("SELECT id FROM drop_packet_arp_info " +
            "WHERE opcode=? AND sender_ip=? AND sender_mac=? AND target_ip=? AND target_mac=?")
        checkErr(err)

        _, err = stmt.Exec(op, sip, smac, tip, tmac)
        checkErr(err)

        err = stmt.QueryRow(op, sip, smac, tip, tmac).Scan(&id)
        if (err == nil) {
            /* update exist one */
            stmt, err := db.Prepare("UPDATE drop_packet_arp_info " +
            "SET last_detected_time=?,drop_id=? where id=?")
            checkErr(err)

            _, err = stmt.Exec(timestamp, drop_index, id)
            checkErr(err)
        } else {
            stmt, err := db.Prepare("INSERT drop_packet_arp_info SET " +
                "opcode=?,sender_ip=?,sender_mac=?,target_ip=?,target_mac=?," +
                "last_detected_time=?,drop_id=?")
            checkErr(err)

            _, err = stmt.Exec(op, sip, smac, tip, tmac, timestamp, drop_index)
            checkErr(err)
        }

        stmt.Close()
    }

    return op, sip, smac, tip, tmac
}

func updateDropPacketIpv4Info(db *sql.DB, packet gopacket.Packet,
    timestamp string, drop_index int) (string, string, string, uint8, uint8, uint16) {
    var id int
    var sip string
    var dip string
    var proto string
    var tos uint8
    var ttl uint8
    var length uint16

    // Get the IPv4 layer from this packet
    if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
        // Get actual ipv4 data from this layer
        ip4, _ := ipLayer.(*layers.IPv4)
        sip = ip4.SrcIP.String()
        dip = ip4.DstIP.String()
        tos = ip4.TOS
        length = ip4.Length
        ttl = ip4.TTL
        proto = ip4.Protocol.String()

        stmt, err := db.Prepare("SELECT id FROM drop_packet_ipv4_info " +
            "WHERE source_ip=? AND destination_ip=? AND protocol=?")
        checkErr(err)

        _, err = stmt.Exec(sip, dip, proto)
        checkErr(err)

        err = stmt.QueryRow(sip, dip, proto).Scan(&id)
        if (err == nil) {
            /* update exist one */
            stmt, err := db.Prepare("UPDATE drop_packet_ipv4_info " +
            "SET tos=?,length=?,ttl=?,last_detected_time=?,drop_id=? WHERE id=?")
            checkErr(err)

            _, err = stmt.Exec(tos, length, ttl, timestamp, drop_index, id)
            checkErr(err)
        } else {
            /* insert new one */
            stmt, err := db.Prepare("INSERT drop_packet_ipv4_info SET " +
                "source_ip=?,destination_ip=?,tos=?,length=?,ttl=?," +
                "protocol=?,last_detected_time=?,drop_id=?")
            checkErr(err)

            _, err = stmt.Exec(sip, dip, tos, length, ttl, proto, timestamp, drop_index)
            checkErr(err)
        }

        stmt.Close()
    }

    return sip, dip, proto, tos, ttl, length
}

func updateDropPacketProtocolInfo(db *sql.DB, packet gopacket.Packet,
    timestamp string, drop_index int) (string, string) {
    var sport string
    var dport string

    // Get the TCP layer from this packet
    if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
        // Get actual TCP data from this layer
        tcp, _ := tcpLayer.(*layers.TCP)
        sport = tcp.SrcPort.String()
        dport = tcp.DstPort.String()
    } else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
        // Get actual UDP data from this layer
        udp, _ := udpLayer.(*layers.UDP)
        sport = udp.SrcPort.String()
        dport = udp.DstPort.String()
    } else {
        return sport, dport
    }

    stmt, err := db.Prepare("SELECT id FROM drop_packet_ip_protocol_info " +
        "WHERE l4_source_port=? AND l4_destination_port=?")
    checkErr(err)

    _, err = stmt.Exec(sport, dport)
    checkErr(err)

    var id int
    err = stmt.QueryRow(sport, dport).Scan(&id)
    if (err == nil) {
        /* update exist one */
        stmt, err := db.Prepare("UPDATE drop_packet_ip_protocol_info " +
            "SET l4_source_port=?,l4_destination_port=?,last_detected_time=?,drop_id=? WHERE id=?")
        checkErr(err)

        _, err = stmt.Exec(sport, dport, timestamp, drop_index, id)
        checkErr(err)
    } else {
        /* insert new one */
        stmt, err := db.Prepare("INSERT drop_packet_ip_protocol_info SET " +
            "l4_source_port=?,l4_destination_port=?,last_detected_time=?,drop_id=?")
        checkErr(err)

        _, err = stmt.Exec(sport, dport, timestamp, drop_index)
        checkErr(err)
    }

    stmt.Close()

    return sport, dport
}

func updateDropPacketDetail(db *sql.DB, device string, ingress_interface string,
    egress_interface string, drop_reason_string string,
    dmac string, smac string, length uint64, etype string, vlanid uint64,
    op uint16, sender_ip string, sender_mac string, target_ip string, target_mac string,
    sip string, dip string, proto string, tos uint8, ttl uint8, iplen uint16,
    l4_sport string, l4_dport string, timestamp string) {

    stmt, err := db.Prepare("SELECT id FROM drop_packet_detail_info " +
        "WHERE device=? AND ingress_interface=? AND egress_interface=? AND drop_reason_string=? " +
        "AND dmac=? AND smac=? AND ether_type=? AND vlanid=? " +
        "AND opcode=? AND sender_ip=? AND sender_mac=? AND target_ip=? AND target_mac=? " +
        "AND source_ip=? AND destination_ip=? AND protocol=? " +
        "AND l4_source_port=? AND l4_destination_port=?")
    checkErr(err)

    _, err = stmt.Exec(device, ingress_interface, egress_interface,
        drop_reason_string, dmac, smac, etype, vlanid,
        op, sender_ip, sender_mac, target_ip, target_mac,
        sip, dip, proto, l4_sport, l4_dport)
    checkErr(err)

    var id int
    err = stmt.QueryRow(device, ingress_interface, egress_interface, drop_reason_string,
        dmac, smac, etype, vlanid,
        op, sender_ip, sender_mac, target_ip, target_mac,
        sip, dip, proto, l4_sport, l4_dport).Scan(&id)
    if (err == nil) {
        /* update exist one */
        stmt, err := db.Prepare("UPDATE drop_packet_detail_info " +
            "SET length=?,tos=?,ttl=?,ip_length=?,last_detected_time=? WHERE id=?")
        checkErr(err)

        _, err = stmt.Exec(length, tos, ttl, iplen, timestamp, id)
        checkErr(err)
    } else {
        /* insert new one */
        stmt, err := db.Prepare("INSERT drop_packet_detail_info SET " +
            "device=?,ingress_interface=?,egress_interface=?,drop_reason_string=?," +
            "dmac=?,smac=?,length=?,ether_type=?,vlanid=?,opcode=?,sender_ip=?,sender_mac=?," +
            "target_ip=?,target_mac=?,source_ip=?,destination_ip=?,protocol=?,tos=?,ttl=?,ip_length=?," +
            "l4_source_port=?,l4_destination_port=?,last_detected_time=?")
        checkErr(err)

        _, err = stmt.Exec(device, ingress_interface, egress_interface, drop_reason_string,
            dmac, smac, length, etype, vlanid,
            op, sender_ip, sender_mac, target_ip, target_mac,
            sip, dip, proto, tos, ttl, iplen, l4_sport, l4_dport, timestamp)
        checkErr(err)
    }

    stmt.Close()
}

func processDropPacket(packet string, addr *net.UDPAddr) {
    ingress_interface := getElement(packet, "ingressPhysicalPort")
    egress_interface := getElement(packet, "egressPhysicalPort")
    drop_reason_string := getElement(packet, "dropReasonString")
    timestamp := getElement(packet, "timeStamp")
    vlanid, _ := strconv.ParseUint(getElement(packet, "vlanId"), 0, 64)
    drop_reason, _ := strconv.ParseInt(getElement(packet, "dropReason"), 0, 64)
    packetsize, _ := strconv.ParseUint(getElement(packet, "packetSize"), 0, 64)
    datasize, _ := strconv.ParseInt(getElement(packet, "dataSize"), 0, 64)
    data := decodePacketData(getElement(packet, "data"))

    db, err := sql.Open("mysql", DbUrl)
    checkErr(err)

    if datasize != 0 {
        pkt := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

        drop_index := updateDropPacketMeta(db, fmt.Sprintf("%s", addr.IP),
            ingress_interface, egress_interface, drop_reason, drop_reason_string, timestamp)
        dmac, smac, etype  := updateDropPacketEthernetInfo(db, pkt, packetsize, timestamp, drop_index)
        e1, vid := updateDropPacketVlanInfo(db, pkt, etype, uint16(vlanid), timestamp, drop_index)
        if (0 != vid) {
            etype = e1
            vlanid = uint64(vid)
        }
        op, sender_ip, sender_mac, target_ip, target_mac := updateDropPacketArpInfo(
            db, pkt, timestamp, drop_index)
        sip, dip, proto, tos, ttl, iplen := updateDropPacketIpv4Info(db, pkt, timestamp, drop_index)
        l4_sport, l4_dport := updateDropPacketProtocolInfo(db, pkt, timestamp, drop_index)
        updateDropPacketDetail(db, fmt.Sprintf("%s", addr.IP),
            ingress_interface, egress_interface, drop_reason_string,
            dmac, smac, packetsize, etype, vlanid,
            op, sender_ip, sender_mac, target_ip, target_mac,
            sip, dip, proto, tos, ttl, iplen,
            l4_sport, l4_dport, timestamp)
    } else {
        _ = updateDropPacketMeta(db, fmt.Sprintf("%s", addr.IP),
            ingress_interface,egress_interface, drop_reason, drop_reason_string,
            timestamp)
    }

    db.Close()
}

func handleClient(conn *net.UDPConn) {
    var buf [1024]byte
    _, addr, err := conn.ReadFromUDP(buf[0:])
    if err != nil {
        return
    }

    packet := string(buf[0:])
    processDropPacket(packet, addr)

    daytime := time.Now().String()
    conn.WriteToUDP([]byte(daytime), addr)
}

func checkErr(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error ", err.Error())
        os.Exit(1)
    }
}

func main() {
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkErr(err)
    conn, err := net.ListenUDP("udp", udpAddr)
    checkErr(err)
    for {
        handleClient(conn)
    }
}

