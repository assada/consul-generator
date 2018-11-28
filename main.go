package main

import "os"

func main() {
	cli := NewCli(os.Stdout, os.Stderr)
	os.Exit(cli.Run(os.Args))
}
