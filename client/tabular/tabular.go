package tabular

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/msg"
)

type formatter struct {
	// TODO
}

func (f formatter) Name() string {
	return "tabular"
}

func (f formatter) Error(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
}

func (f formatter) Response(resp msg.Response) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
	for _, line := range resp.Body {
		noTab := true
		for _, word := range line {
			if noTab {
				noTab = false
			} else {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, word)
		}
		fmt.Fprint(w, "\n")
	}
	w.Flush()
}

func (f formatter) HelpSingleOperation(op client.Operation) {
	header, footer := op.HelpHeaderAndFooter()
	// Summary
	sdesc := op.DescribeShort()
	fmt.Println("Usage:", os.Args[0], sdesc.Cmd, sdesc.First, sdesc.Second)
	// Header
	fmt.Printf("\n%s\n", header)
	// Describe required task name(s), if any
	if tdesc := op.Parser().TaskDescription(); tdesc != "" {
		fmt.Printf("\nRequired task information\n\t%s\n", tdesc)
	}
	// Parameter description
	if pdesc := op.Parser().ParamDescription(); len(pdesc) > 0 {
		fmt.Printf("\nPossible parameters\n")
		w := tabwriter.NewWriter(os.Stdout, 4, 4, 2, ' ', 0)
		for _, par := range pdesc {
			fmt.Fprintf(w, "    %s\t%s\t%s\n",
				par.ParamName, par.ParamValues, par.ParamExplanation)
		}
		// Ignore error, there's nothing we can do if we can't print to the user
		w.Flush()
	}
	// Footer
	if footer != "" {
		fmt.Printf("\n%s\n", footer)
	}
}

func (f formatter) HelpAllOperations(descriptions []argparse.Description) {
	fmt.Printf("\nUsage: %s [command] <task(s)> <parameters>\n\n", os.Args[0])
	fmt.Println("Available commands")

	w := tabwriter.NewWriter(os.Stdout, 4, 4, 2, ' ', 0)
	for _, descr := range descriptions {
		fmt.Fprintf(w, "    %s\t%s\t%s\t%s\n", descr.Cmd, descr.First, descr.Second, descr.What)
	}
	w.Flush()
}

func init() {
	client.RegisterFormatter(formatter{})
}
