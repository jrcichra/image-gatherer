package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/jrcichra/image-gatherer/pkg/config"
	"github.com/jrcichra/image-gatherer/pkg/plugin"
	"github.com/sourcegraph/conc"
)

func main() {
	configFile := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// load the configuration file
	c := config.LoadConfigOrDie(*configFile)
	var g conc.WaitGroup

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
		g.Go(func() {
			name, entry := name, entry // scoping
			var p plugin.InputPlugin
			switch entry.PluginName {
			case "git":
				p = &plugin.Git{}
			case "semver":
				p = &plugin.Semver{}
			default:
				log.Printf("unknown plugin: %s", entry.PluginName)
			}
			digest, err := p.GetTag(ctx, entry.Name, entry.Options)
			if err != nil {
				log.Printf("%s err: %v", name, err)
			}
			log.Printf("%s: %s", name, digest)
			outp.Add(name, digest)
		})
	}
	g.Wait()
	// handle the output
	if err := outp.Synth(ctx, c.Output.Options); err != nil {
		log.Fatalln(err)
	}
	log.Println("success!")
}
