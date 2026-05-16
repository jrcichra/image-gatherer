package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c := config.LoadConfigOrDie(cfg.ConfigFile)
	for {
		slog.Info("starting run")
		if err := run(ctx, c); err != nil {
			slog.Error("run failed", "err", err)
		}
		slog.Info("run complete, sleeping", "interval", cfg.Interval)
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return
		case <-time.After(cfg.Interval):
		}
	}
}

func run(ctx context.Context, c config.Config) error {
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
		name, entry := name, entry
		if entry.Pin != "" {
			slog.Info("pinning container", "name", name, "pin", entry.Pin)
			separator := ":"
			if strings.Contains(entry.Pin, "sha256") {
				separator = "@"
			}
			outp.Add(name, fmt.Sprintf("%s%s%s", entry.Name, separator, entry.Pin))
			continue
		}
		g.Go(func() error {
			var p plugin.InputPlugin
			switch entry.PluginName {
			case "git":
				p = &plugin.Git{}
			case "semver":
				p = &plugin.Semver{}
			default:
				return fmt.Errorf("unknown plugin: %s", entry.PluginName)
			}
			tag, err := p.GetTag(gctx, entry.Name, entry.Options)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			slog.Info("resolved tag", "name", name, "tag", tag)
			outp.Add(name, tag)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return outp.Synth(ctx, c.Output.Options)
}
