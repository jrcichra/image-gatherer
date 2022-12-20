package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jrcichra/latest-image-gatherer/pkg/config"
	"github.com/jrcichra/latest-image-gatherer/pkg/output"

	"golang.org/x/sync/errgroup"
)

func main() {
	// load the configuration file
	c := config.LoadConfigOrDie("config.yml")
	fmt.Println(c)
	// make an errgroup which will run through each container
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	outp := output.NewOutput()
	for name, entry := range c.Entries {
		name, entry := name, entry // scoping
		g.Go(func() error {
			name, entry := name, entry // scoping
			switch {
			case entry.UpdateType == "git":
				digest, err := entry.Git.Get(ctx, entry.Container)
				if err != nil {
					return err
				}
				outp.Add(name, digest)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatalln(err)
	}
	// write out the file
	b, err := outp.Marshal()
	if err != nil {
		log.Fatalln(err)
	}
	if err := os.WriteFile("output.yml", b, 0644); err != nil {
		log.Fatalln(err)
	}
}
