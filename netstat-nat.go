package main

import (
	"github.com/dominikh/simple-router/conntrack"
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/nat"

	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

// TODO implement the following flags
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

func main() {
	flag.Parse()

	var which nat.Flag

	if *onlySNAT {
		which = nat.SNAT
	}

	if *onlyDNAT {
		which = nat.DNAT
	}

	if *onlyLocal {
		which = nat.Local
	}

	if *onlyRouted {
		which = nat.Routed
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

	natFlows := nat.GetNAT(flows, which)
	for _, flow := range natFlows {
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
