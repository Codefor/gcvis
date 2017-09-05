// gcvis is a tool to assist you visualising the operation of
// the go runtime garbage collector.
//
// usage:
//
//     gcvis program [arguments]...
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var iface = flag.String("i", "", "specify interface to use. defaults to all.")
var port = flag.Int("p", 12345, "specify port to use.")
var w = flag.Bool("wait", false, "force to wait")

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage :\n %s <args> your_go_program <your args>...\n or \n cat gogc.log|%s -wait=True\n\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var pipeRead io.ReadCloser
	var subcommand *SubCommand

	flag.Parse()
	if len(flag.Args()) < 1 {
		pipeRead = os.Stdin
	} else {
		subcommand = NewSubCommand(flag.Args())
		pipeRead = subcommand.PipeRead
		go subcommand.Run()
	}

	parser := NewParser(pipeRead)

	title := strings.Join(flag.Args(), " ")
	if len(title) == 0 {
		title = fmt.Sprintf("%s:%d", *iface, *port)
	}

	gcvisGraph := NewGraph(title, GCVIS_TMPL)
	server := NewHttpServer(*iface, *port, &gcvisGraph)

	go parser.Run()
	go server.Start()

	url := server.Url()

	log.Printf("server started on %s", url)

	for {
		select {
		case gcTrace := <-parser.GcChan:
			gcvisGraph.AddGCTraceGraphPoint(gcTrace)
		case scvgTrace := <-parser.ScvgChan:
			gcvisGraph.AddScavengerGraphPoint(scvgTrace)
		case output := <-parser.NoMatchChan:
			fmt.Fprintln(os.Stderr, output)
		case <-parser.done:
			if parser.Err != nil {
				fmt.Fprintf(os.Stderr, parser.Err.Error())
				os.Exit(1)
			}

			fmt.Fprintln(os.Stderr, "parser done")
			os.Exit(0)
		}
	}

	if subcommand != nil && subcommand.Err() != nil {
		fmt.Fprintf(os.Stderr, subcommand.Err().Error())
		os.Exit(1)
	}

	if *w {
		fmt.Fprintf(os.Stderr, "force to wait...")
		var c chan bool
		<-c
	}
}
