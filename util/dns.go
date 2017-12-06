package util

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"log"
	"net"
	"runtime"
)

// Create a DNS response with dns data, pack with udp, and ip header.
func CreateDNSResponse(SrcIP net.IP, SrcPort uint16, DstIP net.IP, DstPort uint16, pkt []byte) []byte {
	ip := &layers.IPv4{
		SrcIP:    SrcIP,
		DstIP:    DstIP,
		Protocol: layers.IPProtocolUDP,
		Version:  uint8(4),
		IHL:      uint8(5),
		TTL:      uint8(64),
	}
	udp := &layers.UDP{SrcPort: layers.UDPPort(SrcPort), DstPort: layers.UDPPort(DstPort)}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		log.Println("SetNetworkLayerForChecksum failed", err)
		return nil
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	gopacket.SerializeLayers(buf, opts,
		ip,
		udp,
		gopacket.Payload(pkt),
	)

	packetData := buf.Bytes()
	return packetData
}

func UpdateDNSServers(setFlag bool) {
	var shell string
	if runtime.GOOS == "darwin" {
		shell = `
function updateDNS {
  services=$(networksetup -listnetworkserviceorder | grep 'Hardware Port')
  while read line; do
      sname=$(echo $line | awk -F  "(, )|(: )|[)]" '{print $2}')
      sdev=$(echo $line | awk -F  "(, )|(: )|[)]" '{print $4}')
      # echo "Current service: $sname, $sdev, $currentservice"
      if [ -n "$sdev" ]; then
          ifout="$(ifconfig $sdev 2>/dev/null)"
          echo "$ifout" | grep 'status: active' > /dev/null 2>&1
          rc="$?"
          if [ "$rc" -eq 0 ]; then
              currentservice="$sname"
              currentdevice="$sdev"
              currentmac=$(echo "$ifout" | awk '/ether/{print $2}')
          fi
      fi
  done <<< "$(echo "$services")"

  if [ -n "$currentservice" ]; then
      echo "Current networkservice is $currentservice"
  else
      >&2 echo "Could not find current service"
      exit 1
  fi

  case "$1" in
    d|default)
      olddns=$(networksetup -getdnsservers "$currentservice")
      echo "old dns is $olddns, set dns to default"
      networksetup -setdnsservers "$currentservice" empty
      ;;
    g|google)
      olddns=$(networksetup -getdnsservers "$currentservice")
      echo "old dns is $olddns, set dns to google dns"
      networksetup -setdnsservers "$currentservice" 8.8.8.8 4.4.4.4
      ;;
    a|ali)
      olddns=$(networksetup -getdnsservers "$currentservice")
      echo "old dns is $olddns, set dns to alidns"
      networksetup -setdnsservers "$currentservice" "223.5.5.5"
      ;;
    l|local)
      olddns=$(networksetup -getdnsservers "$currentservice")
      echo "old dns is $olddns, set dns to 127.0.0.1"
      networksetup -setdnsservers "$currentservice" "127.0.0.1"
      ;;
    *)
      echo "You have failed to specify what to do correctly."
      exit 1
      ;;
  esac
}

function flushCache {
  sudo dscacheutil -flushcache
  sudo killall -HUP mDNSResponder
}
`
	} else if runtime.GOOS == "linux" || runtime.GOOS == "freebsd" {
		shell = `
function updateDNS {
  case "$1" in
    g|google)
      echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
      ;;
    a|ali)
      echo "nameserver 223.5.5.5" | sudo tee /etc/resolv.conf
      ;;
    l|local)
      echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
      ;;
    *)
      echo "You have failed to specify what to do correctly."
      exit 1
      ;;
  esac
}

function flushCache {
  nscd -K
  nscd
}
`
	} else {
		log.Println("Without support for", runtime.GOOS)
		return
	}
	if setFlag {
		shell += `
updateDNS l
`
	} else {
		shell += `
updateDNS a
flushCache
`
	}
	ExecShell(shell)
}
