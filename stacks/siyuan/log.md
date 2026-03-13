# SiYuan

## Sources
- Official repo: https://github.com/siyuan-note/siyuan
- Docker instructions from README.md: https://github.com/siyuan-note/siyuan#docker-hosting

## Notes
- SiYuan is a privacy-first personal knowledge management system with block-level references and Markdown WYSIWYG
- Official image: `b3log/siyuan`
- Compose taken from the Docker Compose example in the official README
- Web UI accessible on port 6806
- `--accessAuthCode` should be set for security (access authorization)
- PUID/PGID control file permission ownership inside the container
- Workspace data persisted via bind mount at /siyuan/workspace
