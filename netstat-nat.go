package main

import (
	"honnef.co/go/conntrack"
	"honnef.co/go/netdb"

	flag "github.com/ogier/pflag"

	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"text/tabwriter"
)

// TODO implement the following flags
//       -N: display NAT box connection information (only valid with SNAT & DNAT)

type FlowSlice conntrack.FlowSlice

type SortBySource struct{ FlowSlice }
type SortByDestination struct{ FlowSlice }
type SortBySPort struct{ FlowSlice }
type SortByDPort struct{ FlowSlice }
type SortByState struct{ FlowSlice }

func (flows FlowSlice) Swap(i, j int) {
	flows[i], flows[j] = flows[j], flows[i]
}

func (flows FlowSlice) Len() int {
	return len(flows)
}

func (flows SortBySource) Less(i, j int) bool {
	return flows.FlowSlice[i].Original.Source.String() < flows.FlowSlice[j].Original.Source.String()
}

func (flows SortByDestination) Less(i, j int) bool {
	return flows.FlowSlice[i].Original.Destination.String() < flows.FlowSlice[j].Original.Destination.String()
}

func (flows SortBySPort) Less(i, j int) bool {
	return flows.FlowSlice[i].Original.SPort < flows.FlowSlice[j].Original.SPort
}

func (flows SortByDPort) Less(i, j int) bool {
	return flows.FlowSlice[i].Original.DPort < flows.FlowSlice[j].Original.DPort
}

func (flows SortByState) Less(i, j int) bool {
	return flows.FlowSlice[i].State < flows.FlowSlice[j].State
}

var Version = "0.1.0"

var onlySNAT = flag.BoolP("snat", "S", false, "Display only SNAT connections")
var onlyDNAT = flag.BoolP("dnat", "D", false, "Display only DNAT connections")
var onlyLocal = flag.BoolP("local", "L", false, "Display only local connections (originating from or going to the router)")
var onlyRouted = flag.BoolP("routed", "R", false, "Display only connections routed through the router")
var noResolve = flag.BoolP("no-resolve", "n", false, "Do not resolve hostnames")
var noHeader = flag.BoolP("no-header", "o", false, "Strip output header")
var protocol = flag.StringP("protocol", "p", "", "Filter connections by protocol")
var sourceHost = flag.StringP("source", "s", "", "Filter by source IP")
var destinationHost = flag.StringP("destination", "d", "", "Filter by destination IP")
var displayVersion = flag.BoolP("version", "v", false, "Print version")
var sortBy = flag.StringP("sort", "r", "src", "Sort connections (src | dst | src-port | dst-port | state)")
var _ = flag.BoolP("extended-hostnames", "x", false, "This flag serves no purpose other than compatibility")

func main() {
	flag.Parse()

	if *displayVersion {
		fmt.Println("Version " + Version)
		os.Exit(0)
	}

	which := conntrack.SNATFilter | conntrack.DNATFilter

	if *onlySNAT {
		which = conntrack.SNATFilter
	}

	if *onlyDNAT {
		which = conntrack.DNATFilter
	}

	if *onlyLocal {
		which = conntrack.LocalFilter
	}

	if *onlyRouted {
		which = conntrack.RoutedFilter
	}

	flows, err := conntrack.Flows()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read conntrack information: %s\n", err.Error())
		os.Exit(1)
	}

	filteredFlows := flows.FilterByType(which)
	if *protocol != "" {
		protoent, ok := netdb.GetProtoByName(*protocol)
		if !ok {
			fmt.Fprintf(os.Stderr, "'%s' is not a known protocol.\n", *protocol)
			os.Exit(1)
		}
		filteredFlows = filteredFlows.FilterByProtocol(protoent)
	}

	if *sourceHost != "" {
		sourceIP := net.ParseIP(*sourceHost) // TODO support hostnames
		filteredFlows = filteredFlows.Filter(func(flow conntrack.Flow) bool {
			return flow.Original.Source.Equal(sourceIP)
		})
	}

	if *destinationHost != "" {
		destinationIP := net.ParseIP(*destinationHost) // TODO support hostnames
		filteredFlows = filteredFlows.Filter(func(flow conntrack.Flow) bool {
			return flow.Original.Destination.Equal(destinationIP)
		})
	}

	switch *sortBy {
	case "src":
		sort.Sort(SortBySource{FlowSlice(filteredFlows)})
	case "dst":
		sort.Sort(SortByDestination{FlowSlice(filteredFlows)})
	case "src-port":
		sort.Sort(SortBySPort{FlowSlice(filteredFlows)})
	case "dst-port":
		sort.Sort(SortByDPort{FlowSlice(filteredFlows)})
	case "state":
		sort.Sort(SortByState{FlowSlice(filteredFlows)})
	}

	tabWriter := &tabwriter.Writer{}
	tabWriter.Init(os.Stdout, 0, 0, 4, ' ', 0)

	if !*noHeader {
		fmt.Fprintln(tabWriter, "Proto\tSource Address\tDestination Address\tState")
	}

	for _, flow := range filteredFlows {
		sHostname := resolve(flow.Original.Source, *noResolve)
		dHostname := resolve(flow.Original.Destination, *noResolve)
		sPortName := portToName(int(flow.Original.SPort), flow.Protocol.Name)
		dPortName := portToName(int(flow.Original.DPort), flow.Protocol.Name)
		fmt.Fprintf(tabWriter, "%s\t%s:%s\t%s:%s\t%s\n",
			flow.Protocol.Name,
			sHostname,
			sPortName,
			dHostname,
			dPortName,
			flow.State,
		)
	}
	tabWriter.Flush()
}

func portToName(port int, protocol string) string {
	servent, ok := netdb.GetServByPort(port, protocol)
	if !ok {
		return strconv.FormatInt(int64(port), 10)
	}

	return servent.Name
}

func resolve(ip net.IP, noop bool) string {
	if noop {
		return ip.String()
	}

	lookup, err := net.LookupAddr(ip.String())
	if err == nil && len(lookup) > 0 {
		return lookup[0]
	}

	return ip.String()
}
