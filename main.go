package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/google/go-github/v43/github"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

func run() error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerologr.NameFieldName = "logger"
	zerologr.NameSeparator = "/"

	zl := zerolog.New(zerolog.NewConsoleWriter())
	zl = zl.With().Caller().Timestamp().Logger()
	var log logr.Logger = zerologr.New(&zl)

	tok := os.Getenv("GITHUB_TOKEN")
	org := "thepwagner-org"
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	packages, _, err := client.Organizations.ListPackages(ctx, org, &github.PackageListOptions{PackageType: github.String("container")})
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}
	log.Info("listed packages", "package_count", len(packages))

	for _, pkg := range packages {
		pkgLog := log.WithValues("package_name", *pkg.Name)
		pkgLog.Info("processing package", "package_id", *pkg.ID, "version_count", pkg.GetVersionCount())
		versions, _, err := client.Organizations.PackageGetAllVersions(ctx, org, "container", *pkg.Name, &github.PackageListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list versions of %q: %w", *pkg.Name, err)
		}
		pkgLog.Info("listed versions", "version_count", len(versions))

	versions:
		for _, v := range versions {
			ctr := v.GetMetadata().GetContainer()
			if ctr == nil {
				continue
			}

			for _, t := range ctr.Tags {
				if t == "latest" {
					continue versions
				}
			}

			if len(ctr.Tags) == 0 {
				pkgLog.Info("deleting untagged version version", "version_id", *v.ID)
				_, err := client.Organizations.PackageDeleteVersion(ctx, org, "container", *pkg.Name, *v.ID)
				if err != nil {
					return fmt.Errorf("failed to delete version %q %d: %w", *pkg.Name, *v.ID, err)
				}
			}
		}
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
