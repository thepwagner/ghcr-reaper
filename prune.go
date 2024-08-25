package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-github/v64/github"
)

type packageClient interface {
	ListPackages(ctx context.Context, org string, opts *github.PackageListOptions) ([]*github.Package, *github.Response, error)
	PackageGetAllVersions(ctx context.Context, org, packageType, packageName string, opts *github.PackageListOptions) ([]*github.PackageVersion, *github.Response, error)
	PackageDeleteVersion(ctx context.Context, org, packageType, packageName string, id int64) (*github.Response, error)
}

var _ packageClient = (*github.OrganizationsService)(nil)
var _ packageClient = (*github.UsersService)(nil)

func prunePackages(ctx context.Context, client packageClient, org string) error {
	packages, _, err := client.ListPackages(ctx, org, &github.PackageListOptions{PackageType: github.String("container")})
	if err != nil {
		return fmt.Errorf("failed to list packages: %w", err)
	}
	slog.Info("listed packages", "package_count", len(packages))

	for _, pkg := range packages {
		pkgLog := slog.Default().With("package_name", *pkg.Name)
		pkgLog.Info("processing package", "package_id", *pkg.ID, "version_count", pkg.GetVersionCount())
		versions, _, err := client.PackageGetAllVersions(ctx, org, "container", *pkg.Name, &github.PackageListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list versions of %q: %w", *pkg.Name, err)
		}
		pkgLog.Info("listed versions", "version_count", len(versions))

		// Initially remove untagged images, and note the :latest digest.
		deleted := map[int64]struct{}{}
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
				deleted[*v.ID] = struct{}{}
			}
		}

		// TODO: not deleting the heads of open PRs would be cool.

		// If :latest was detected, we can delete all other signatures/attestations:
		if latestDigest == "" {
			continue
		}
		digest := strings.Replace(latestDigest, ":", "-", -1)
		latestAtt := fmt.Sprintf("%s.att", digest)
		latestSig := fmt.Sprintf("%s.sig", digest)
	secondPass:
		for _, v := range versions {
			if _, ok := deleted[*v.ID]; ok {
				continue
			}

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
