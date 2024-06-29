package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jrcichra/image-gatherer/pkg/config"
	"github.com/jrcichra/image-gatherer/pkg/plugin"

	"golang.org/x/sync/errgroup"
)

type cfg struct {
	ConfigFile string
	Interval   time.Duration
}

func main() {
	var cfg cfg

	flag.StringVar(&cfg.ConfigFile, "config", "config.yaml", "path to configuration file")
	flag.DurationVar(&cfg.Interval, "interval", time.Minute*5, "interval for runs")
	flag.Parse()

	// load the configuration file
	c := config.LoadConfigOrDie(cfg.ConfigFile)
	for {
		log.Println("starting run...")
		if err := run(c); err != nil {
			log.Println("run failed:", err)
		}
		log.Printf("run complete. Sleeping %s before next run\n", cfg.Interval.String())
		time.Sleep(cfg.Interval)
	}
}

func run(c config.Config) error {
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
		return fmt.Errorf("unknown output plugin: %s", c.Output.PluginName)
	}

	for name, entry := range c.Containers {
		name, entry := name, entry // scoping
		if entry.Pin != "" {
			log.Printf("pinning %s to %s. Skipping collection", name, entry.Pin)
			separator := ":"
			if strings.Contains(entry.Pin, "sha256") {
				separator = "@"
			}
			outp.Add(name, fmt.Sprintf("%s%s%s", entry.Name, separator, entry.Pin))
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
		return err
	}
	// handle the output
	if err := outp.Synth(ctx, c.Output.Options); err != nil {
		return err
	}
	return nil
}
