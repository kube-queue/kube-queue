package main

import (
	"flag"
	"log"

	"github.com/kube-queue/kube-queue/cmd/default-permission-counter/app/options"
	app "github.com/kube-queue/kube-queue/cmd/default-permission-counter/app/server"
)

func main() {
	s := options.NewServerOption()
	s.AddFlags(flag.CommandLine)

	flag.Parse()

	if err := app.Run(s); err != nil {
		log.Fatalln(err)
	}
}
