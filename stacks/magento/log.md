# Magento Open Source Stack

## Research
- No official Docker image from magento/magento2 repo
- No docker-compose.yml in the upstream repo
- Used Bitnami's well-maintained Magento image (`bitnami/magento`)
- Services: Magento (PHP app), MariaDB 10.6, Elasticsearch 7
- Bitnami images are widely used for Magento deployments

## Changes from upstream
- Created compose.yaml based on Bitnami's recommended configuration
- All services use Bitnami images for consistency
- Environment variables with sensible defaults in .env
