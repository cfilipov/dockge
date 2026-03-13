# ThingsBoard

## Source
- GitHub: https://github.com/thingsboard/thingsboard
- Docker Hub: thingsboard/tb-postgres

## Description
Open-source IoT platform for device management, data collection, processing, and visualization. Supports MQTT, CoAP, and HTTP protocols for device connectivity.

## Stack
- **thingsboard**: Core platform (HTTP on 8080, MQTT on 1883, CoAP on 5683/UDP)
- **postgres**: PostgreSQL database backend
- In-memory queue by default (can switch to Kafka for production)
