# GarageHQ

## Sources
- Quick start docs: https://garagehq.deuxfleurs.fr/documentation/quick-start/
- Configuration reference: https://garagehq.deuxfleurs.fr/documentation/reference-manual/configuration/
- Docker Hub: https://hub.docker.com/r/dxflrs/garage

## Notes
- Docker run command from official quick start guide converted to compose format
- Image: `dxflrs/garage:v2.2.0`
- Ports: 3900 (S3 API), 3901 (RPC), 3902 (S3 Web), 3903 (Admin)
- Config file (garage.toml) bind-mounted as required by the container
- replication_factor set to 1 for single-node setup
