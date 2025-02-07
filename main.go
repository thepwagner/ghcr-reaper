package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/lmittmann/tint"
	"golang.org/x/oauth2"
)

func run(ctx context.Context) error {
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
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		panic(err)
	}
}
