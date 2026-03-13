# WebHook Tester

## Source
- Repository: https://github.com/tarampampam/webhook-tester
- README: https://raw.githubusercontent.com/tarampampam/webhook-tester/master/README.md

## Notes
- No docker-compose file in the repository
- Converted from docker run command in README: `docker run --rm -t -p "8080:8080/tcp" ghcr.io/tarampampam/webhook-tester:2`
- Also available on Docker Hub as tarampampam/webhook-tester
- Environment variables from README documentation
- Supports memory, redis, and filesystem storage drivers; defaulting to memory for simplicity
- Single-service stack, no dependencies required for basic usage
