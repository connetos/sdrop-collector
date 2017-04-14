do
    local p_sdrop = Proto("sdrop", "Streaming drop packet and drop reason");

    local sdrop_meta_ingress_port = ProtoField.string("sdrop.inport", "Ingress Physical Port", base.NONE)
    local sdrop_meta_egress_port = ProtoField.string("sdrop.outport", "Egress Physical Port", base.NONE)
    local sdrop_meta_vlan_id = ProtoField.string("sdrop.vlanid", "Assigned VLAN ID", base.DEC)
    local sdrop_meta_drop_reason_str = ProtoField.string("sdrop.dropreasonstr", "Drop Reason", base.NONE)
    local sdrop_meta_time_stamp = ProtoField.string("sdrop.stamp", "Last Detected Time", base.NONE)
    local sdrop_meta_packet_size = ProtoField.string("sdrop.pktsize", "Oringinal Packet Length", base.DEC)
    local sdrop_meta_data_size = ProtoField.string("sdrop.datasize", "Sampled Packet Length", base.DEC)

    p_sdrop.fields = {
        sdrop_meta_ingress_port,
        sdrop_meta_egress_port,
        sdrop_meta_vlan_id,
        sdrop_meta_drop_reason,
		sdrop_meta_drop_reason_str,
        sdrop_meta_time_stamp,
        sdrop_meta_packet_size,
        sdrop_meta_data_size,
    }
	
    local function get_element(str, key)
        local pattern = "<"..key..">(.*)</"..key..">"
        for w in string.gmatch(str, pattern) do
            return w
        end
    end

    function p_sdrop.dissector(buf, pinfo, root)
        local payload = buf(0, buf:len() - 1)
		local raw_pkt = get_element(payload:string(), "data")        		
		local datasize = get_element(payload:string(), "dataSize")
        local pktsize = get_element(payload:string(), "packetSize")
        local timestamp = get_element(payload:string(), "timeStamp")
		local dropreasonstr = get_element(payload:string(), "dropReasonString")
        local vlanid = get_element(payload:string(), "vlanId")
        local inport = get_element(payload:string(), "ingressPhysicalPort")
        local outport = get_element(payload:string(), "egressPhysicalPort")
	
	    local s1,s2 = string.find(payload:string(), "<data>")
		local e1,e2 = string.find(payload:string(), "</data>")
	
        local sdrop_tree = root:add(p_sdrop, buf:range(offset, s1))
        sdrop_tree:add(sdrop_meta_ingress_port, inport)
        sdrop_tree:add(sdrop_meta_egress_port, outport)
        sdrop_tree:add(sdrop_meta_vlan_id, vlanid)
		sdrop_tree:add(sdrop_meta_drop_reason_str, dropreasonstr)
        sdrop_tree:add(sdrop_meta_time_stamp, timestamp)
		sdrop_tree:add(sdrop_meta_data_size, datasize)
		sdrop_tree:add(sdrop_meta_packet_size, pktsize)

		local eth_dis = Dissector.get("eth_withoutfcs")
		local b = ByteArray.new(raw_pkt)
		local buf_frame = ByteArray.tvb(b, "Raw Payload")
		eth_dis:call(buf_frame, pinfo, root)
    end

    local udp_encap_table = DissectorTable.get("udp.port")
    udp_encap_table:add(32768, p_sdrop)
end
