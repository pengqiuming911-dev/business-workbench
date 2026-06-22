# GitHub Actions ECS Deploy

This repository can now auto-deploy to Alibaba Cloud ECS on every push to `main`.

## Files

- `.github/workflows/deploy.yml`
- `scripts/deploy/remote-deploy.sh`

## Server prerequisites

Prepare the ECS host once:

1. Install `nginx`, `systemd`, `curl`.
2. Create a deploy user with sudo access.
3. Create the app directory:

```bash
mkdir -p /var/www/business-workbench/shared
mkdir -p /var/www/business-workbench/releases
```

4. Make sure the deploy user can write `/var/www/business-workbench`.
5. Make sure the deploy user can run `sudo systemctl` and `sudo nginx -t`.
6. Make sure nginx serves `/var/www/business-workbench/frontend/dist` or follows the `frontend` symlink under `/var/www/business-workbench`.

## GitHub secrets

Add these repository secrets:

- `SERVER_HOST`
- `SERVER_USER`
- `SERVER_PORT`
- `SSH_PRIVATE_KEY`
- `FRONTEND_URL`
- `FEISHU_APP_ID`
- `FEISHU_APP_SECRET`
- `FEISHU_REDIRECT_URI`
- `DEEPSEEK_API_KEY`
- `DEEPSEEK_API_URL`
- `DEEPSEEK_MODEL`
- `FEISHU_PUSH_WEBHOOK`
- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_SECURE`
- `SMTP_USER`
- `SMTP_PASS`
- `SMTP_FROM`

Note:

- `FEISHU_REDIRECT_URI` and `FRONTEND_URL` are preserved from the existing server-side `${APP_DIR}/shared/.env` on each deploy when that file already exists.
- This prevents a successful production deployment from silently reverting the Feishu callback URL back to an older value.

## What deployment does

1. Build the frontend with Vite.
2. Build the Go backend for Linux amd64.
3. Package a release tarball.
4. Upload the tarball and remote deploy script to the ECS host.
5. Write `${APP_DIR}/shared/.env`, preserving existing `FEISHU_REDIRECT_URI` and `FRONTEND_URL` when present.
6. Unpack into `${APP_DIR}/releases/<release-name>`.
7. Link shared `.env` and `data.sqlite`.
8. Switch `${APP_DIR}/current` to the new release.
9. Refresh `${APP_DIR}/frontend` and `${APP_DIR}/backend-go` symlinks for nginx and operator compatibility.
10. Restart `business-workbench` with systemd.
11. Verify `http://127.0.0.1:3001/api/health`.

## Rollback behavior

If the backend health check fails after restart, the remote script switches `current` back to the previous release and tries to restart the service again.

## Trigger

Any push to `main` triggers deployment automatically.
