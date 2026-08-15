package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/service"
	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/store"
	"github.com/zhangchengcheng/go-cc-0006-allocation-planner/internal/transport"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader, output io.Writer) error {
	flags := flag.NewFlagSet("planner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	action := flags.String("action", "allocate", "allocate or report")
	if err := flags.Parse(args); err != nil {
		return err
	}

	request, err := transport.DecodeRequest(input)
	if err != nil {
		return err
	}

	inventory := store.NewMemoryInventory(request.Inventory)
	audit := store.NewMemoryAudit()
	planner := service.NewPlanner(inventory, audit)

	var response any
	switch *action {
	case "allocate":
		allocation, err := planner.Allocate(context.Background(), request.Order)
		if err != nil {
			return err
		}
		response = allocation
	case "report":
		report, err := planner.Report(context.Background())
		if err != nil {
			return err
		}
		response = report
	default:
		return fmt.Errorf("unknown action %q", *action)
	}

	return json.NewEncoder(output).Encode(response)
}
