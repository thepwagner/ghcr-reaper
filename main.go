package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/go-github/v57/github"
	"github.com/lmittmann/tint"

	"golang.org/x/oauth2"
)

func run() error {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{})))

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC3339,
		}),
	))

	tok, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		return fmt.Errorf("GITHUB_TOKEN not set")
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	if err := prunePackages(ctx, client.Users, "thepwagner"); err != nil {
		return err
	}
	if err := prunePackages(ctx, client.Organizations, "thepwagner-org"); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
