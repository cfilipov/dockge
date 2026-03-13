# Collabora Online Development Edition (CODE)

## Sources
- Docker Hub image: `collabora/code`
- Official Docker docs: https://sdk.collaboraonline.com/docs/installation/CODE_Docker_image.html
- GitHub repo: https://github.com/CollaboraOnline/online (no compose file in repo)

## Notes
- Port 9980 is the default HTTP/HTTPS port
- `--privileged` flag recommended for faster jail creation via bind mount
- `aliasgroup1` defines allowed WOPI hosts (needed for integration with Nextcloud, etc.)
- SSL disabled via `extra_params` for local/dev use; enable for production
- Admin console accessible at https://localhost:9980/browser/dist/admin/admin.html
