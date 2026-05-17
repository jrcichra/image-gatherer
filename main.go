package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jrcichra/image-gatherer/pkg/config"
	"github.com/jrcichra/image-gatherer/pkg/plugin"
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

	if err := validateConfig(c); err != nil {
		slog.Error("invalid config", "err", err)
		os.Exit(1)
	}

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

func validateConfig(c config.Config) error {
	if _, err := plugin.NewOutputPlugin(c.Output.PluginName, c.Output.Options); err != nil {
		return fmt.Errorf("output: %w", err)
	}
	for name, entry := range c.Containers {
		if entry.Pin != "" {
			continue
		}
		if _, err := plugin.NewInputPlugin(entry.PluginName, entry.Options); err != nil {
			return fmt.Errorf("container %s: %w", name, err)
		}
	}
	return nil
}

func run(ctx context.Context, c config.Config) error {
	outp, err := plugin.NewOutputPlugin(c.Output.PluginName, c.Output.Options)
	if err != nil {
		return err
	}

	if err := outp.Open(ctx, c.Output.Options); err != nil {
		return fmt.Errorf("failed to open output: %w", err)
	}

	var wg sync.WaitGroup
	for name, entry := range c.Containers {
		name, entry := name, entry
		if entry.Pin != "" {
			slog.Info("pinning container", "name", name, "pin", entry.Pin)
			separator := ":"
			if strings.Contains(entry.Pin, "sha256") {
				separator = "@"
			}
			// pins always override whatever was loaded from the previous output
			outp.Add(name, fmt.Sprintf("%s%s%s", entry.Name, separator, entry.Pin))
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			p, err := plugin.NewInputPlugin(entry.PluginName, entry.Options)
			if err != nil {
				slog.Error("failed to create input plugin, skipping", "name", name, "err", err)
				return
			}
			tag, err := p.GetTag(ctx, entry.Name, entry.Options)
			if err != nil {
				slog.Error("failed to resolve tag, skipping", "name", name, "err", err)
				return
			}
			slog.Info("resolved tag", "name", name, "tag", tag)
			outp.Add(name, tag)
		}()
	}
	wg.Wait()
	return outp.Close(ctx, c.Output.Options)
}
