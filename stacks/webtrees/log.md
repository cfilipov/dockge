# webtrees — Research Log

## Result: DONE

Created compose.yaml based on the official docker-compose.yml from NathanVaughn/webtrees-docker (495k+ pulls). Used variable substitution for passwords via .env file.

webtrees is the web's leading online collaborative genealogy application, allowing users to view and edit genealogy data from any browser.

## Services
- **app**: webtrees web application (ghcr.io/nathanvaughn/webtrees)
- **db**: MariaDB database

## Sources
1. https://github.com/fisharebest/webtrees — official repo, no Docker support
2. Docker Hub API search — found nathanvaughn/webtrees (495k pulls), dtjs48jkt/webtrees (1.4M pulls)
3. https://github.com/NathanVaughn/webtrees-docker — well-maintained Docker wrapper
4. https://raw.githubusercontent.com/NathanVaughn/webtrees-docker/master/docker-compose.yml — official compose example (used as source)
