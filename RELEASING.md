# Releasing

Releases are source-only GitHub releases identified by annotated Semantic
Versioning tags. The module has no generated release artifacts.

## Checklist

1. Start from a clean branch based on the current `main`.
2. Move the intended entries from `Unreleased` to a dated version in
   `CHANGELOG.md`, and update its comparison links.
3. Run the local preflight with the current stable Go release:

   ```sh
   make ci
   make bench
   ```

4. Merge the release pull request only after every required GitHub check is
   green.
5. Update local `main`, verify its commit, and create an annotated tag:

   ```sh
   git switch main
   git pull --ff-only origin main
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

6. Create a GitHub Release for the existing tag. Summarize user-visible API,
   compatibility, correctness, and security changes from the changelog.
7. Verify the published module through the public Go proxy:

   ```sh
   GOPROXY=https://proxy.golang.org go list -m github.com/ruzcash/go-zcashblob@vX.Y.Z
   ```

8. Verify the version page on
   [pkg.go.dev](https://pkg.go.dev/github.com/ruzcash/go-zcashblob), the GitHub
   Release link in `CHANGELOG.md`, and a clean install from a temporary module.

Repository controls are part of the release gate: private vulnerability
reporting must remain enabled, and `main` must require the aggregate CI and
vulnerability-scan checks with force pushes and deletions disabled.
