# Deployment Runbook

## Preferred Production Flow

For routine production releases, prefer local builds and artifact upload.
Do not run full frontend or Go compilation on the server unless there is no
practical fallback.

Recommended flow:

1. Build the frontend locally with `bun run build` in `web/`.
2. Build the backend locally for Linux amd64 with the embedded `web/dist`.
3. Upload the prebuilt binary to the server.
4. Replace `/new-api` inside the running `app` container with `docker cp`.
5. Restart only the `app` container and verify `/api/status` and `/`.

This keeps server CPU usage low and avoids repeated `docker build` work on the
production host.

Example local build:

```powershell
Set-Location web
bun run build
Set-Location ..

$version = (Get-Content VERSION -Raw).Trim()
$ldflags = if ($version) {
  "-s -w -X `"github.com/QuantumNous/new-api/common.Version=$version`""
} else {
  "-s -w"
}

$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags $ldflags -o .tmp/new-api-linux-amd64
```

Example upload and in-container replacement:

```bash
scp -i .ssh/opc/ssh-key-2025-04-28.key .tmp/new-api-linux-amd64 \
  akwang10000@34.64.84.172:/tmp/new-api-linux-amd64

ssh -i .ssh/opc/ssh-key-2025-04-28.key akwang10000@34.64.84.172 '
  docker cp new-api-app-1:/new-api /tmp/new-api-backup &&
  docker cp /tmp/new-api-linux-amd64 new-api-app-1:/new-api &&
  docker restart new-api-app-1 &&
  sleep 5 &&
  curl -fsS http://127.0.0.1:3000/api/status &&
  curl -fsS https://routeropenai.xyz/api/status
'
```

If the new binary fails, restore the backup binary into the container before
restarting it again.

## Historical ARM64 Build Note

This note is kept for an earlier ARM64 deployment incident. If a future
deployment targets an ARM64 host and still uses on-server Docker builds, pass
the target platform explicitly:

```bash
docker build \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=arm64 \
  -t new-api-local:<tag>-arm64 .
```

Do not rely on the Dockerfile defaults for production builds. The current
Dockerfile defaults `TARGETARCH` to `amd64` when no build argument is provided.
That can produce an amd64 `/new-api` binary inside an image that is then started
on the ARM64 host.

## 2026-04-18 Deployment Incident

During deployment of commit `d608411a`, the first image was built without
`TARGETARCH=arm64`. The container started and then exited immediately with:

```text
exec /new-api: exec format error
```

Observed symptoms:

- `docker logs new-api-current` showed `exec /new-api: exec format error`.
- Local server health check to `http://127.0.0.1:3001/api/status` failed with
  connection refused.
- Public access temporarily returned `502 Bad Gateway` while the new container
  was not running.

Recovery performed:

```bash
docker rm -f new-api-current
docker rename new-api-previous-d608411a new-api-current
docker start new-api-current
```

After service recovery, the image was rebuilt with explicit ARM64 build args:

```bash
docker build \
  --build-arg TARGETOS=linux \
  --build-arg TARGETARCH=arm64 \
  -t new-api-local:20260418-d608411a-arm64 .
```

The new image was then redeployed using the existing production data volume and
environment variables:

```bash
docker run -d --name new-api-current \
  -p 3001:3000 \
  -v /home/opc/apps/new-api-data:/data \
  -e CHATWOOT_HMAC_TOKEN='<redacted>' \
  new-api-local:20260418-d608411a-arm64
```

Post-deploy checks completed successfully:

- `http://127.0.0.1:3001/api/status`
- `http://127.0.0.1:3001/`
- `http://129.146.121.209:3001/`
- `https://routeropenai.xyz/`

Keep the previous container stopped until the new deployment has been verified:

```bash
docker ps -a --filter name='new-api-'
```
