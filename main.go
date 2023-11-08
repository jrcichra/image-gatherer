package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/jrcichra/image-gatherer/pkg/config"
	"github.com/jrcichra/image-gatherer/pkg/plugin"

	"golang.org/x/sync/errgroup"
)

func main() {
	configFile := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	// load the configuration file
	c := config.LoadConfigOrDie(*configFile)
	// make an errgroup which will run through each container
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g, gctx := errgroup.WithContext(ctx)

	var outp plugin.OutputPlugin
	switch c.Output.PluginName {
	case "file":
		outp = &plugin.File{}
	case "git":
		outp = &plugin.Git{}
	default:
		log.Fatalf("unknown output plugin: %s", c.Output.PluginName)
	}

	for name, entry := range c.Containers {
		name, entry := name, entry // scoping
		if entry.Pin != "" {
			log.Printf("pinning %s to %s. Skipping collection", name, entry.Pin)
			outp.Add(name, entry.Pin)
			continue
		}
		g.Go(func() error {
			name, entry := name, entry // scoping
			var p plugin.InputPlugin
			switch entry.PluginName {
			case "git":
				p = &plugin.Git{}
			case "semver":
				p = &plugin.Semver{}
			default:
				return fmt.Errorf("unknown plugin: %s", entry.PluginName)
			}
			digest, err := p.GetTag(gctx, entry.Name, entry.Options)
			if err != nil {
				return fmt.Errorf("%s err: %v", name, err)
			}
			log.Printf("%s: %s", name, digest)
			outp.Add(name, digest)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatalln(err)
	}
	// handle the output
	if err := outp.Synth(ctx, c.Output.Options); err != nil {
		log.Fatalln(err)
	}
	log.Println("success!")
}
