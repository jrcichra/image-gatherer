package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jrcichra/latest-image-gatherer/pkg/config"
	"github.com/jrcichra/latest-image-gatherer/pkg/output"
	"github.com/jrcichra/latest-image-gatherer/pkg/plugin"

	"golang.org/x/sync/errgroup"
)

func main() {
	// load the configuration file
	c := config.LoadConfigOrDie("config.yml")
	// make an errgroup which will run through each container
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, ctx := errgroup.WithContext(ctx)

	outp := output.NewOutput()
	for name, entry := range c.Entries {
		name, entry := name, entry // scoping
		g.Go(func() error {
			name, entry := name, entry // scoping
			var (
				image string
				tag   string
				err   error
			)
			switch entry.UpdateType {
			case "git":
				image, tag, err = entry.Git.Get(ctx, entry.Container)
			case "semver":
				var s plugin.Semver
				image, tag, err = s.Get(ctx, entry.Container)
			default:
				return fmt.Errorf("unknown update type: %s", entry.UpdateType)
			}
			if err != nil {
				return fmt.Errorf("%s err: %v", name, err)
			}
			log.Printf("%s matched tag: %s", name, tag)
			outp.Add(name, image)
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
