# PostHog Reverse Proxy

Nginx reverse proxy for PostHog analytics. Routes all PostHog traffic through your own domain to bypass ad blockers and keep analytics requests first-party.

## Running

```bash
docker build -t fluid-reverse-proxy .
docker run -d -p 8080:8080 --name fluid-reverse-proxy fluid-reverse-proxy
```

Verify it's running:

```bash
curl http://localhost:8080/
# => ok
```

## Directing Traffic

Point your PostHog client's API host at the proxy instead of `https://app.posthog.com`.

**JavaScript SDK:**

```js
posthog.init('<your-project-key>', {
  api_host: 'http://localhost:8080',
})
```

**In production**, replace `http://localhost:8080` with your deployed proxy URL (e.g. `https://proxy.yourdomain.com`).

## How It Works

All requests are proxied to `app.posthog.com`. The proxy:

- Strips and re-adds CORS headers to avoid duplicates
- Rate limits ingestion endpoints (`/i/`, `/batch/`) to 10 req/s per IP with burst of 20
- Exposes a health check at `GET /` (exact match only)
