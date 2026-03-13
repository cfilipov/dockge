# Apache Airflow
**Project:** https://airflow.apache.org/
**Source:** https://github.com/apache/airflow
**Status:** done
**Compose source:** https://airflow.apache.org/docs/apache-airflow/3.1.8/docker-compose.yaml
## What was done
- Used official docker-compose from Airflow docs (CeleryExecutor setup)
- Simplified to core services: postgres, redis, apiserver, scheduler, worker, triggerer, init
- Created bind-mount directories: dags, logs, config, plugins
- Default credentials: airflow/airflow
## Issues
- None
