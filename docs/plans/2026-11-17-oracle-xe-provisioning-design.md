# Oracle XE provisioning and database selection

## Problem

The installed faststart image contains a complete seed database in its image layer. Mounting an empty host directory at `data_dir` hides it. Initialization then leaves only `dbconfig/XE`, which the image treats as an existing database. Startup fails with ORA-00205 and `Restart=always` repeats the failure.

The link prompt also skips database selection when a project lists any service, even Redis alone. Oracle external and local choices need distinct labels and an explicit terminal confirmation when selecting Oracle.

## Design

Use a generic, preset-driven `data_seed` declaration rather than database names in the copy mechanism. Copy the image data into a sibling staging directory, validate required nonempty files, then rename it into place. Reject incomplete existing data. Keep valid databases unchanged. A Linux ExecCondition checks the same state before boot starts; explicit CLI, UI and MCP service starts also run the preparation.

Changing to a slow-initializing image avoids the hidden seed but changes the default startup cost. Named Podman volumes copy image data automatically but change the storage and recovery model. Seeding the existing host directory preserves lerd's current data layout.

Keep local Oracle XE as a service preset and external Oracle as a connection configuration. Use preset metadata for the local display label. Prompt when no database has been selected, even if non-database services are configured. Selecting external Oracle in the terminal asks whether to use local XE, defaulting to preserving the external choice.

## Validation

Test fresh seeding, reuse without overwriting data, incomplete existing data, interrupted copy and retry, missing files in the image, invalid relative paths, cleanup of image-reader containers, boot conditions, link prompt eligibility and Oracle location choices. Build the UI and Go binary, run the Go tests and vet, then exercise a real image using isolated data and a temporary Oracle container.

## Validation evidence

- UI production build, the Go `nogui` build, `go vet -tags nogui ./...`, formatting and diff checks passed. The WSL host lacks the tray's appindicator development library, so Go validation uses the project's supported `CGO_ENABLED=0` fallback.
- Fourteen new tests cover the seed lifecycle, image-copy cleanup, boot condition, Oracle location choice, existing connection preservation and the complete env flow, including a failed service start.
- A temporary Oracle 21-slim-faststart instance opened the seeded database and LERDPDB. A test row with value 42 survived a graceful restart. The temporary container and data were removed after testing; the original paused Oracle was unchanged.
- A transient systemd service with deliberately incomplete data was skipped by ExecCondition, with zero restarts. A second preparation of valid data returned without copying it again.
- The unfiltered Go suite has four baseline failures, reproduced from the clean fork commit 705dd21f: `TestForwarderPortFree_IPv6HostBracketedCorrectly`, `TestS3ListBucketsReportsObjectsAndSize`, `TestS3CreateBucketOpensItToAnonymousClients` and `TestS3RemoveBucketEmptiesItFirst`. They are outside the Oracle change. Local validation logs remain under the ignored `build/` directory.
- Existing uncommitted work in the main checkout was preserved. The installed binary and original paused Oracle data were not changed.

## Release preparation

The next fork release is `v1.35.0-oracle.1`. Its changelog, README version and release notes are prepared with this change. The release workflow starts on a pushed `v*` tag and publishes the binaries through GoReleaser. Do not create the tag until the PR has merged into the fork's default branch, `oracle-oci8-support`, and the full release gate is green. The four baseline failures listed above still block declaring the local gate green.
