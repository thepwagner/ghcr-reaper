package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/google/go-github/v51/github"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

func run() error {
	log := logger()

	tok, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok {
		return fmt.Errorf("GITHUB_TOKEN not set")
	}

	ctx := context.Background()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	if err := prunePackages(ctx, log, client.Users, "thepwagner"); err != nil {
		return err
	}
	if err := prunePackages(ctx, log, client.Organizations, "thepwagner-org"); err != nil {
		return err
	}
	return nil
}

func logger() logr.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(zerolog.NewConsoleWriter())
	zl = zl.With().Caller().Timestamp().Logger()
	return zerologr.New(&zl)
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
