# Deployment Runbook

## ARM64 Server Docker Build

Production runs on an ARM64 server. When building the Docker image on the server,
pass the target platform explicitly:

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
