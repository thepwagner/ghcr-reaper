package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/google/go-github/v43/github"
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

	if err := prunePackages(ctx, log, client.Organizations, "thepwagner-org"); err != nil {
		return err
	}
	if err := prunePackages(ctx, log, client.Users, "thepwagner"); err != nil {
		return err
	}
	return nil
}

type packageClient interface {
	ListPackages(ctx context.Context, org string, opts *github.PackageListOptions) ([]*github.Package, *github.Response, error)
	PackageGetAllVersions(ctx context.Context, org, packageType, packageName string, opts *github.PackageListOptions) ([]*github.PackageVersion, *github.Response, error)
	PackageDeleteVersion(ctx context.Context, org, packageType, packageName string, id int64) (*github.Response, error)
}

var _ packageClient = (*github.OrganizationsService)(nil)
var _ packageClient = (*github.UsersService)(nil)

func prunePackages(ctx context.Context, log logr.Logger, client packageClient, org string) error {
	packages, _, err := client.ListPackages(ctx, org, &github.PackageListOptions{PackageType: github.String("container")})
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}
	log.Info("listed packages", "package_count", len(packages))

	for _, pkg := range packages {
		pkgLog := log.WithValues("package_name", *pkg.Name)
		pkgLog.Info("processing package", "package_id", *pkg.ID, "version_count", pkg.GetVersionCount())
		versions, _, err := client.PackageGetAllVersions(ctx, org, "container", *pkg.Name, &github.PackageListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list versions of %q: %w", *pkg.Name, err)
		}
		pkgLog.Info("listed versions", "version_count", len(versions))

		var latestDigest string
	firstPass:
		for _, v := range versions {
			ctr := v.GetMetadata().GetContainer()
			if ctr == nil {
				continue
			}

			for _, t := range ctr.Tags {
				if t == "latest" {
					latestDigest = *v.Name
					continue firstPass
				}
			}

			if len(ctr.Tags) == 0 {
				pkgLog.Info("deleting untagged version", "version_id", *v.ID)
				_, err := client.PackageDeleteVersion(ctx, org, "container", *pkg.Name, *v.ID)
				if err != nil {
					return fmt.Errorf("failed to delete version %q %d: %w", *pkg.Name, *v.ID, err)
				}
			}
		}

		if latestDigest == "" {
			continue
		}

		versions, _, err = client.PackageGetAllVersions(ctx, org, "container", *pkg.Name, &github.PackageListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list versions of %q: %w", *pkg.Name, err)
		}
		pkgLog.Info("listed versions", "version_count", len(versions))

		digest := strings.Replace(latestDigest, ":", "-", -1)
		latestAtt := fmt.Sprintf("%s.att", digest)
		latestSig := fmt.Sprintf("%s.sig", digest)
	secondPass:
		for _, v := range versions {
			ctr := v.GetMetadata().GetContainer()
			if ctr == nil {
				continue
			}

			for _, t := range ctr.Tags {
				switch t {
				case latestAtt, latestSig, "latest":
					continue secondPass
				}
			}

			pkgLog.Info("deleting stale version", "version_id", *v.ID)
			_, err := client.PackageDeleteVersion(ctx, org, "container", *pkg.Name, *v.ID)
			if err != nil {
				return fmt.Errorf("failed to delete version %q %d: %w", *pkg.Name, *v.ID, err)
			}
		}
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
