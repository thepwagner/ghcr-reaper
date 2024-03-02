# ghcr-reaper

Tool to clean old images from my GitHub Container Registry.

1. Iterate through public/private containers here:
* https://github.com/thepwagner?ecosystem=container&tab=packages
* https://github.com/orgs/thepwagner-org/packages?ecosystem=container

2. Delete every image except the `:latest`.
3. Delete any signatures or attestations associated with the deleted images.
