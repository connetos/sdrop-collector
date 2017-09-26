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
    username = "root"
    password = "1q2w3e"
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
    timestamp string, layer2_index int, tag_index int, arp_index int,
    layer3_index int, layer4_index int) {

    // if old drop packet meta exists
    stmt, err := db.Prepare("SELECT id,sample_packets FROM drop_packet_meta_info " +
        "where device=? AND ingress_interface=? AND egress_interface=? AND drop_reason=?")
    checkErr(err)

    _, err = stmt.Exec(device, ingress_interface, egress_interface, drop_reason)
    checkErr(err)

    var id int
    var sample_packets int
    err = stmt.QueryRow(device, ingress_interface, egress_interface, drop_reason).Scan(&id, &sample_packets)
    if (err == nil) {
        fmt.Println("update meta", id)

        /* update exist one */
        stmt, err := db.Prepare("update drop_packet_meta_info " +
            "set sample_packets=?,last_detected_time=?,layer2_index=?,tag_index=?," +
            "arp_index=?,layer3_index=?,layer4_index=? where id=?")
        checkErr(err)

        res, err := stmt.Exec(sample_packets + 1, timestamp, layer2_index, tag_index,
            arp_index, layer3_index, layer4_index, id)
        checkErr(err)

        _, err = res.RowsAffected()
        checkErr(err)
    } else {
        /* insert new one */
        fmt.Println("meta not found")

        stmt, err := db.Prepare(
            "INSERT drop_packet_meta_info SET " +
            "device=?,ingress_interface=?,egress_interface=?,drop_reason=?," +
            "drop_reason_string=?,last_detected_time=?," +
            "sample_packets=?,layer2_index=?,tag_index=?,arp_index=?,layer3_index=?,layer4_index=?")
        checkErr(err)

        res, err := stmt.Exec(device, ingress_interface, egress_interface,
            drop_reason, drop_reason_string, timestamp, 1, layer2_index,
            tag_index, arp_index, layer3_index, layer4_index)
        checkErr(err)

        id, err := res.LastInsertId()
        checkErr(err)

        fmt.Println(id)
    }

    stmt.Close();
}

func updateDropPacketEthernetInfo(db *sql.DB, packet gopacket.Packet, timestamp string) int {
    var id int

    // Get the ethernet layer from this packet
    if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
        // Get actual ethernet data from this layer
        eth, _ := ethLayer.(*layers.Ethernet)
        smac := eth.SrcMAC.String()
        dmac := eth.DstMAC.String()
        etype := eth.EthernetType
        length := eth.Length

        fmt.Printf("From src %s to dst %s\n", smac, dmac)

        // if old drop packet meta exists
        stmt, err := db.Prepare("SELECT id FROM drop_packet_layer2_info " +
            "where smac=? AND dmac=? AND ether_type=?")
        checkErr(err)

        _, err = stmt.Exec(smac, dmac, etype)
        checkErr(err)

        err = stmt.QueryRow(smac, dmac, etype).Scan(&id)
        if (err == nil) {
            fmt.Println("update", id)

            /* update exist one */
            stmt, err := db.Prepare("update drop_packet_layer2_info " +
            "set length=?,last_detected_time=? where id=?")
            checkErr(err)

            res, err := stmt.Exec(length, timestamp, id)
            checkErr(err)

            _, err = res.RowsAffected()
            checkErr(err)
        } else {
            /* insert new one */
            fmt.Println("not found")

            stmt, err := db.Prepare("INSERT drop_packet_layer2_info SET " +
                "smac=?,dmac=?,ether_type=?,length=?,last_detected_time=?")
            checkErr(err)

            res, err := stmt.Exec(smac, dmac, etype, length, timestamp)
            checkErr(err)

            id, err := res.LastInsertId()
            checkErr(err)

            fmt.Println(id)
        }

        stmt.Close();
    }

    return id
}

func updateDropPacketVlanInfo(db *sql.DB, packet gopacket.Packet, timestamp string) int {
    var id int

    // Get the dot1q layer from this packet
    if dot1qLayer := packet.Layer(layers.LayerTypeDot1Q); dot1qLayer != nil {
        // Get actual dot1q data from this layer
        tag, _ := dot1qLayer.(*layers.Dot1Q)
        fmt.Printf("From vlan %d\n", tag.VLANIdentifier)

        vlanid := tag.VLANIdentifier
        etype := tag.Type

        // if old drop packet meta exists
        stmt, err := db.Prepare("SELECT id FROM drop_packet_vlan_tag_info " +
        "where tpid=? AND vlanid=?")
        checkErr(err)

        _, err = stmt.Exec(etype, vlanid)
        checkErr(err)

        err = stmt.QueryRow(etype, vlanid).Scan(&id)
        if (err == nil) {
            fmt.Println("update", id)

            /* update exist one */
            stmt, err := db.Prepare("update drop_packet_vlan_tag_info " +
            "set last_detected_time=? where id=?")
            checkErr(err)

            res, err := stmt.Exec(timestamp, id)
            checkErr(err)

            _, err = res.RowsAffected()
            checkErr(err)
        } else {
            /* insert new one */
            fmt.Println("not found")

            stmt, err := db.Prepare("INSERT drop_packet_vlan_tag_info SET " +
                "tpid=?,vlanid=?,last_detected_time=?")
            checkErr(err)

            res, err := stmt.Exec(etype, vlanid, timestamp)
            checkErr(err)

            id, err := res.LastInsertId()
            checkErr(err)

            fmt.Println(id)
        }

        stmt.Close()
    }

    return id
}

func updateDropPacketArpInfo(db *sql.DB, packet gopacket.Packet, timestamp string) int {
    var id int

    // Get the arp layer from this packet
    if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
        // Get actual arp data from this layer
        arp, _ := arpLayer.(*layers.ARP)
        fmt.Printf("From src %s to dst %s\n", arp.SourceHwAddress, arp.DstHwAddress)

        sip := IPToString(arp.SourceProtAddress)
        smac := macToString(arp.SourceHwAddress)
        tip := IPToString(arp.DstProtAddress)
        tmac := macToString(arp.DstHwAddress)
        op := arp.Operation

        // if old drop packet meta exists
        stmt, err := db.Prepare("SELECT id FROM drop_packet_arp_info " +
        "where opcode=? AND sender_ip=? AND sender_mac=? AND target_ip=? AND target_mac=?")
        checkErr(err)

        _, err = stmt.Exec(op, sip, smac, tip, tmac)
        checkErr(err)

        err = stmt.QueryRow(op, sip, smac, tip, tmac).Scan(&id)
        if (err == nil) {
            fmt.Println("update", id)

            /* update exist one */
            stmt, err := db.Prepare("update drop_packet_arp_info " +
            "set last_detected_time=? where id=?")
            checkErr(err)

            res, err := stmt.Exec(timestamp, id)
            checkErr(err)

            _, err = res.RowsAffected()
            checkErr(err)
        } else {
            /* insert new one */
            fmt.Println("not found")

            stmt, err := db.Prepare("INSERT drop_packet_arp_info SET " +
                "opcode=?,sender_ip=?,sender_mac=?,target_ip=?,target_mac=?," +
                "last_detected_time=?")
            checkErr(err)

            res, err := stmt.Exec(op, sip, smac, tip, tmac, timestamp)
            checkErr(err)

            id, err := res.LastInsertId()
            checkErr(err)

            fmt.Println(id)
        }

        stmt.Close()
    }

    return id
}

func updateDropPacketIpv4Info(db *sql.DB, packet gopacket.Packet, timestamp string) int {
    var id int

    // Get the IPv4 layer from this packet
    if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
        // Get actual ipv4 data from this layer
        ip4, _ := ipLayer.(*layers.IPv4)
        fmt.Printf("From src %s to dst %s\n", ip4.SrcIP, ip4.DstIP)

        sip := ip4.SrcIP.String()
        dip := ip4.DstIP.String()
        tos := ip4.TOS
        length := ip4.Length
        ttl := ip4.TTL
        proto := ip4.Protocol

        // if old drop packet meta exists
        stmt, err := db.Prepare("SELECT id FROM drop_packet_ipv4_info " +
            "where source_ip=? AND destination_ip=? AND protocol=?")
        checkErr(err)

        _, err = stmt.Exec(sip, dip, proto)
        checkErr(err)

        err = stmt.QueryRow(sip, dip, proto).Scan(&id)
        if (err == nil) {
            fmt.Println("update", id)

            /* update exist one */
            stmt, err := db.Prepare("update drop_packet_ipv4_info " +
            "set tos=?,length=?,ttl=?,last_detected_time=? where id=?")
            checkErr(err)

            res, err := stmt.Exec(tos, length, ttl, timestamp, id)
            checkErr(err)

            _, err = res.RowsAffected()
            checkErr(err)
        } else {
            /* insert new one */
            fmt.Println("not found")

            stmt, err := db.Prepare("INSERT drop_packet_ipv4_info SET " +
                "source_ip=?,destination_ip=?,tos=?,length=?,ttl=?," +
                "protocol=?,last_detected_time=?")
            checkErr(err)

            res, err := stmt.Exec(sip, dip, tos, length, ttl, proto, timestamp)
            checkErr(err)

            id, err := res.LastInsertId()
            checkErr(err)

            fmt.Println(id)
        }

        stmt.Close()
    }

    return id
}

func updateDropPacketProtocolInfo(db *sql.DB, packet gopacket.Packet, timestamp string) int {
    var sport string
    var dport string
    // Get the TCP layer from this packet
    if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
        // Get actual TCP data from this layer
        tcp, _ := tcpLayer.(*layers.TCP)
        fmt.Printf("From src port %d to dst port %d\n", tcp.SrcPort, tcp.DstPort)

        sport = tcp.SrcPort.String()
        dport = tcp.DstPort.String()
    } else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
        // Get actual UDP data from this layer
        udp, _ := udpLayer.(*layers.UDP)
        fmt.Printf("From src port %d to dst port %d\n", udp.SrcPort, udp.DstPort)

        sport = udp.SrcPort.String()
        dport = udp.DstPort.String()
    } else {
        return 0
    }

    // if old drop packet meta exists
    stmt, err := db.Prepare("SELECT id FROM drop_packet_ip_protocol_info " +
        "where l4_source_port=? AND l4_destination_port=?")
    checkErr(err)

    _, err = stmt.Exec(sport, dport)
    checkErr(err)

    var id int
    err = stmt.QueryRow(sport, dport).Scan(&id)
    if (err == nil) {
        fmt.Println("update", id)

        /* update exist one */
        stmt, err := db.Prepare("update drop_packet_ip_protocol_info " +
            "set l4_source_port=?,l4_destination_port=?,last_detected_time=? where id=?")
        checkErr(err)

        res, err := stmt.Exec(sport, dport, timestamp, id)
        checkErr(err)

        _, err = res.RowsAffected()
        checkErr(err)
    } else {
        /* insert new one */
        fmt.Println("not found")

        stmt, err := db.Prepare("INSERT drop_packet_ip_protocol_info SET " +
            "l4_source_port=?,l4_destination_port=?,last_detected_time=?")
        checkErr(err)

        res, err := stmt.Exec(sport, dport, timestamp)
        checkErr(err)

        id, err := res.LastInsertId()
        checkErr(err)

        fmt.Println(id)
    }

    stmt.Close()

    return id
}

func processDropPacket(packet string, addr *net.UDPAddr) {
    ingress_interface := getElement(packet, "ingressPhysicalPort")
    egress_interface := getElement(packet, "egressPhysicalPort")
    drop_reason_string := getElement(packet, "dropReasonString")
    timestamp := getElement(packet, "timeStamp")
    vlanid, _ := strconv.ParseInt(getElement(packet, "vlanId"), 0, 64)
    drop_reason, _ := strconv.ParseInt(getElement(packet, "dropReason"), 0, 64)
    datasize, _ := strconv.ParseInt(getElement(packet, "dataSize"), 0, 64)
    packetsize, _ := strconv.ParseInt(getElement(packet, "packetSize"), 0, 64)
    data := decodePacketData(getElement(packet, "data"))

    fmt.Printf("remote address %s\n", addr.IP)
    fmt.Printf("ingress interface %s\n", ingress_interface)
    fmt.Printf("egress interface %s\n", egress_interface)
    fmt.Printf("vlan id %d\n", vlanid)
    fmt.Printf("drop reason %d\n", drop_reason)
    fmt.Printf("drop reason %s\n", drop_reason_string)
    fmt.Printf("timestamp %s\n", timestamp)
    fmt.Printf("packetsize %d\n", packetsize)
    fmt.Printf("data size %d\n", datasize)
    //fmt.Printf("data %s\n", data)
    fmt.Printf("\n")

    db, err := sql.Open("mysql", DbUrl)
    checkErr(err)

    if datasize != 0 {
        pkt := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.Default)

        layer2_index := updateDropPacketEthernetInfo(db, pkt, timestamp)
        tag_index := updateDropPacketVlanInfo(db, pkt, timestamp)
        arp_index := updateDropPacketArpInfo(db, pkt, timestamp)
        layer3_index := updateDropPacketIpv4Info(db, pkt, timestamp)
        layer4_index := updateDropPacketProtocolInfo(db, pkt, timestamp)
        updateDropPacketMeta(db, fmt.Sprintf("%s", addr.IP),
            ingress_interface, egress_interface, drop_reason, drop_reason_string,
            timestamp, layer2_index, tag_index, arp_index, layer3_index, layer4_index)
    } else {
        updateDropPacketMeta(db, fmt.Sprintf("%s", addr.IP),
            ingress_interface,egress_interface, drop_reason, drop_reason_string,
            timestamp, 0, 0, 0, 0, 0)
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
    service := ":32768"
    udpAddr, err := net.ResolveUDPAddr("udp4", service)
    checkErr(err)
    conn, err := net.ListenUDP("udp", udpAddr)
    checkErr(err)
    for {
        handleClient(conn)
    }
}

