# sdrop-collector

This repository contains information related to sDrop.

wireshark-plugin contain wireshark plugin to display the export information of sDrop.

Now sDrop uses xml to encap the information of the dropped packet, below is an example.

``` 
<?xml version="1.0" encoding="ISO-8859-1"?>
<dropPacket>
    <model version="1.0" author="ConnetOS"/>
    <metaData>
        <ingressPhysicalPort>te-1/1/1</ingressPhysicalPort>
        <egressPhysicalPort>NA</egressPhysicalPort>
        <vlanId>52</vlanId>
        <dropReason>2</dropReason>
        <dropReasonString>Tag Vlan Not Exist</dropReasonString>
        <timeStamp>2017-04-07 20:07:41</timeStamp>
        <packetSize>157</packetSize>
        <dataSize>128</dataSize>
    </metaData>
    <data>
        2C600C7BC1FB000000BBBB44810000340800450C0087000040004006B605373737140B0B0B0A2410008000000000000000005000FFFF8B6F000001010008010200000000123000001231000012320000123300001234000012300000123000001230000012310000123200001233000012340000123100001232000012330000
    </data>
</dropPacket>
```

