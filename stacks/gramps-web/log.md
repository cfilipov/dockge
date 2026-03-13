# Gramps Web — Research Log

## Result: DONE

Created compose.yaml from the official Gramps Web documentation examples.

Gramps Web is a web-based frontend for the Gramps genealogy system, enabling collaborative family tree editing from any browser. It uses a Celery worker for background tasks and Redis as the message broker.

## Services
- **grampsweb**: Main web application (ghcr.io/gramps-project/grampsweb)
- **grampsweb_celery**: Background task worker (same image, different command)
- **grampsweb_redis**: Redis message broker and cache

## Sources
1. https://github.com/gramps-project/gramps-web — main repo, has Dockerfiles
2. https://github.com/gramps-project/gramps-web-docs — documentation repo
3. https://raw.githubusercontent.com/gramps-project/gramps-web-docs/main/examples/docker-compose-base/docker-compose.yml — official base compose example (used as source)
