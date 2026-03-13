# GeneWeb — Research Log

## Result: DONE

Created compose.yaml using the community Docker image `schuellerf/geneweb` (500k+ pulls on Docker Hub).

GeneWeb is an open-source genealogy software with a web interface, written in OCaml. The official repo has a Dockerfile but no published image. The community image by schuellerf is the most established Docker option.

## Services
- **geneweb**: Main application (port 2317) + setup interface (port 2316)

## Sources
1. https://github.com/geneweb/geneweb — official repo, has Dockerfile but no published image
2. Docker Hub API search — found `schuellerf/geneweb` (506k pulls, active)
3. https://github.com/schuellerf/docker-geneweb — newer version repo with docker run instructions
4. Docker Hub full_description for schuellerf/geneweb — docker run command with ports, volumes, env vars
