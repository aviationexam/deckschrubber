# syntax=docker/dockerfile:1.7
#
# Build deckschrubber from a published module version via `go install`.
#
# The module version is supplied at build time via `DECKSCHRUBBER_VERSION`
# (defaults to `latest` for local tinkering). CI passes the GitHub Release
# tag name (e.g. `v0.9.0`) so each published image is reproducible and
# matches the corresponding release exactly.
#
# Base images are pinned to alpine minor tags so Dependabot opens a PR
# on each alpine minor bump while Docker continues to auto-roll patch
# releases under the pinned tag. The builder alpine minor is kept in
# lockstep with the runtime alpine minor to avoid musl/CA-bundle drift
# between stages.
#
# Base images are referenced as literal `FROM` tags rather than via an
# `ARG` indirection because Dependabot's docker ecosystem file-updater
# does not rewrite `FROM ${VAR}` references (see
# dependabot/dependabot-core#4597, #4837 - closed unmerged). Keeping
# the tags inline ensures Dependabot can actually bump them.

FROM golang:1.26-alpine3.23 AS build

# Override at build time to pin a specific release, e.g.
#   docker build --build-arg DECKSCHRUBBER_VERSION=v0.9.0 .
ARG DECKSCHRUBBER_VERSION=latest
ARG DECKSCHRUBBER_MODULE=github.com/aviationexam/deckschrubber

ENV CGO_ENABLED=0 \
    GOFLAGS=-trimpath \
    GOBIN=/out

# `go install` falls back to direct VCS (git ls-remote + git clone) when
# proxy.golang.org returns a cache miss for the requested version. Fresh
# tags typically hit this path for the first few minutes after `git push
# origin <tag>` because the proxy indexes lazily, so every Release
# workflow run that fires immediately after tagging races the proxy.
# Without `git` on PATH the fallback aborts with "exec: \"git\": executable
# file not found in $PATH" (bit v0.9.1 and v0.10.1). Installing git in the
# builder stage makes the direct-VCS fallback actually work so the Release
# workflow is independent of proxy indexing latency.
RUN apk add --no-cache git

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go install "${DECKSCHRUBBER_MODULE}@${DECKSCHRUBBER_VERSION}"

FROM alpine:3.23

# ca-certificates is required so deckschrubber can talk to HTTPS registries.
RUN apk add --no-cache ca-certificates

COPY --from=build /out/deckschrubber /usr/local/bin/deckschrubber

# OCI labels populate the GHCR package page (source link, version, license).
# The release workflow additionally applies labels from docker/metadata-action,
# but duplicating the essentials here keeps locally-built images identifiable.
LABEL org.opencontainers.image.title="deckschrubber" \
      org.opencontainers.image.description="Garbage-collect images from a Docker Distribution registry" \
      org.opencontainers.image.source="https://github.com/aviationexam/deckschrubber" \
      org.opencontainers.image.licenses="AGPL-3.0-only"

ENTRYPOINT ["deckschrubber"]
