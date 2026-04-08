---
summary: "Release checklist for gogcli (Bilance GitHub releases)"
---

# Releasing `gogcli`

This playbook is for the Bilance fork and treats GitHub release assets as the source of truth for installs.
Always do **all** steps below: CI, changelog, tag, GitHub release assets, and release verification.

Shortcut scripts (preferred, keep notes non-empty):
```sh
scripts/release.sh X.Y.Z
scripts/verify-release.sh X.Y.Z
```

Assumptions:
- Repo: `bilancetech/gogcli`
- GitHub Releases are the install source for Bilance tooling.

## 0) Prereqs
- Clean working tree on `main`.
- Go toolchain installed (Go version comes from `go.mod`).
- `make` works locally.

## 1) Verify build is green
```sh
make ci
```

Confirm GitHub Actions `ci` is green for the commit you’re tagging:
```sh
gh run list -L 5 --branch main
```

## 2) Update changelog
- Update `CHANGELOG.md` for the version you’re releasing.

Example heading:
- `## 0.1.0 - 2025-12-12`

## 3) Commit, tag & push
```sh
git checkout main
git pull

# commit changelog + any release tweaks
git commit -am "release: vX.Y.Z"

git tag -a vX.Y.Z -m "Release X.Y.Z"
git push origin main --tags
```

## 4) Verify GitHub release artifacts
The tag push triggers `.github/workflows/release.yml` (GoReleaser). Ensure it completes successfully and the release has assets.

```sh
gh run list -L 5 --workflow release.yml
gh release view vX.Y.Z --repo bilancetech/gogcli
```

Ensure GitHub release notes are not empty (mirror the changelog section).

If the workflow needs a rerun:
```sh
gh workflow run release.yml -f tag=vX.Y.Z
```

## 5) Sanity-check GitHub release assets
Verify that the expected platform assets exist and that `checksums.txt` includes all shipped archives:

```sh
gh release download vX.Y.Z --repo bilancetech/gogcli -p checksums.txt -D /tmp/gogcli-release
cat /tmp/gogcli-release/checksums.txt
```

Expected archives:
- `gogcli_X.Y.Z_darwin_amd64.tar.gz`
- `gogcli_X.Y.Z_darwin_arm64.tar.gz`
- `gogcli_X.Y.Z_linux_amd64.tar.gz`
- `gogcli_X.Y.Z_linux_arm64.tar.gz`
- `gogcli_X.Y.Z_windows_amd64.zip`
- `gogcli_X.Y.Z_windows_arm64.zip`

Optional smoke test on the current machine:

```sh
gh release download vX.Y.Z --repo bilancetech/gogcli -p 'gogcli_X.Y.Z_*' -D /tmp/gogcli-release
```

## Notes
- `gog --version` / `gog version` should report the release version from the tag.
- Bilance monorepo setup installs from GitHub Releases, so a release without assets is incomplete.
