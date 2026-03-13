# MongooseIM

## Source
- Repository: https://github.com/esl/MongooseIM
- Docker repo: https://github.com/esl/mongooseim-docker
- Docker image: erlangsolutions/mongooseim (Docker Hub)

## Research
- No docker-compose.yml in main repo
- README references Docker Hub image and separate mongooseim-docker repo
- mongooseim-docker README has docker run examples with port and volume info
- Composed from docker run examples: `-p 5222:5222`, hostname, and Mnesia data volume

## Ports
- 5222: XMPP client connections
- 5269: XMPP server federation
- 5280: HTTP listener
