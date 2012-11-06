package main

import (
	"github.com/dominikh/simple-router/conntrack"

	"flag"
	"fmt"
	"net"
	"os"
	"text/tabwriter"
)

//       -n: don't resolve host/portnames
//       -p <protocol>        : display connections by protocol
//       -s <source-host>     : display connections by source
//       -d <destination-host>: display connections by destination
//       -x: extended hostnames view
//       -r src | dst | src-port | dst-port | state : sort connections
//       -N: display NAT box connection information (only valid with SNAT & DNAT)
//       -v: print version

var onlySNAT = flag.Bool("S", false, "Display only SNAT connections")
var onlyDNAT = flag.Bool("D", false, "Display only DNAT connections")
var onlyLocal = flag.Bool("L", false, "Display only local connections (originating from or going to the router)")
var onlyRouted = flag.Bool("R", false, "Display only connections routed through the router")
var noResolve = flag.Bool("n", false, "Do not resolve hostnames") // TODO resolve port names as well
var noHeader = flag.Bool("o", false, "Strip output header")

var (
	displaySNAT   bool = true
	displayDNAT   bool = true
	displayLocal  bool = false
	displayRouted bool = false
)

var localIPs = make([]*net.IPNet, 0)

func isLocalIP(ip net.IP) bool {
	for _, localIP := range localIPs {
		if localIP.IP.Equal(ip) {
			return true
		}
	}

	return false
}

func init() {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, address := range addresses {
		localIPs = append(localIPs, address.(*net.IPNet))
	}
}

func main() {
	flag.Parse()

	if *onlySNAT {
		displaySNAT = true
		displayDNAT = false
		displayLocal = false
		displayRouted = false
	}

	if *onlyDNAT {
		displaySNAT = false
		displayDNAT = true
		displayLocal = false
		displayRouted = false
	}

	if *onlyLocal {
		displaySNAT = false
		displayDNAT = false
		displayLocal = true
		displayRouted = false
	}

	if *onlyRouted {
		displaySNAT = false
		displayDNAT = false
		displayLocal = false
		displayRouted = true
	}

	flows, err := conntrack.Flows()
	if err != nil {
		panic(err)
	}

	tabWriter := &tabwriter.Writer{}
	tabWriter.Init(os.Stdout, 0, 0, 4, ' ', 0)

	if !*noHeader {
		fmt.Fprintln(tabWriter, "Proto\tSource Address\tDestination Address\tState")
	}

	for _, flow := range flows {
		if (displaySNAT && isSNAT(flow)) ||
			(displayDNAT && isDNAT(flow)) ||
			(displayLocal && isLocal(flow)) ||
			(displayRouted && isRouted(flow)) {

			sHostname := resolve(flow.Original.Source)
			dHostname := resolve(flow.Original.Destination)

			fmt.Fprintf(tabWriter, "%s\t%s:%d\t%s:%d\t%s\n",
				flow.Protocol,
				sHostname,
				flow.Original.SPort,
				dHostname,
				flow.Original.DPort,
				flow.State,
			)
		}
	}
	tabWriter.Flush()
}

func resolve(ip net.IP) string {
	if *noResolve {
		return ip.String()
	}

	lookup, err := net.LookupAddr(ip.String())
	if err == nil && len(lookup) > 0 {
		return lookup[0]
	}

	return ip.String()
}

func isSNAT(flow conntrack.Flow) bool {
	// SNATed flows should reply to our WAN IP, not a LAN IP.
	if flow.Original.Source.Equal(flow.Reply.Destination) {
		return false
	}

	if !flow.Original.Destination.Equal(flow.Reply.Source) {
		return false
	}

	return true
}

func isDNAT(flow conntrack.Flow) bool {
	// Reply must go back to the source; Reply mustn't come from the WAN IP
	if flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) {
		return true
	}

	// Taken straight from original netstat-nat, labelled "DNAT (1 interface)"
	if !flow.Original.Source.Equal(flow.Reply.Source) && !flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) && flow.Original.Destination.Equal(flow.Reply.Destination) {
		return true
	}

	return false
}

func isLocal(flow conntrack.Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// At least one local address
		if isLocalIP(flow.Original.Source) || isLocalIP(flow.Original.Destination) || isLocalIP(flow.Reply.Source) || isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}

func isRouted(flow conntrack.Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// No local addresses
		if !isLocalIP(flow.Original.Source) && !isLocalIP(flow.Original.Destination) && !isLocalIP(flow.Reply.Source) && !isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}
