package main

import (
	"github.com/dominikh/simple-router/conntrack"
	"github.com/dominikh/simple-router/lookup"

	flag "github.com/ogier/pflag"

	"fmt"
	"net"
	"os"
	"text/tabwriter"
)

// TODO implement the following flags
//       -x: extended hostnames view
//       -r src | dst | src-port | dst-port | state : sort connections
//       -N: display NAT box connection information (only valid with SNAT & DNAT)
//       -v: print version

var onlySNAT = flag.BoolP("snat", "S", false, "Display only SNAT connections")
var onlyDNAT = flag.BoolP("dnat", "D", false, "Display only DNAT connections")
var onlyLocal = flag.BoolP("local", "L", false, "Display only local connections (originating from or going to the router)")
var onlyRouted = flag.BoolP("routed", "R", false, "Display only connections routed through the router")
var noResolve = flag.BoolP("no-resolve", "n", false, "Do not resolve hostnames") // TODO resolve port names as well
var noHeader = flag.BoolP("no-header", "o", false, "Strip output header")
var protocol = flag.StringP("protocol", "p", "", "Filter connections by protocol")
var sourceHost = flag.StringP("source", "s", "", "Filter by source IP")
var destinationHost = flag.StringP("destination", "d", "", "Filter by destination IP")

func main() {
	flag.Parse()

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
		panic(err)
	}

	tabWriter := &tabwriter.Writer{}
	tabWriter.Init(os.Stdout, 0, 0, 4, ' ', 0)

	if !*noHeader {
		fmt.Fprintln(tabWriter, "Proto\tSource Address\tDestination Address\tState")
	}

	filteredFlows := flows.FilterByType(which)
	if *protocol != "" {
		filteredFlows = filteredFlows.FilterByProtocol(*protocol)
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

	for _, flow := range filteredFlows {
		sHostname := lookup.Resolve(flow.Original.Source, *noResolve)
		dHostname := lookup.Resolve(flow.Original.Destination, *noResolve)

		fmt.Fprintf(tabWriter, "%s\t%s:%d\t%s:%d\t%s\n",
			flow.Protocol,
			sHostname,
			flow.Original.SPort,
			dHostname,
			flow.Original.DPort,
			flow.State,
		)
	}
	tabWriter.Flush()
}
