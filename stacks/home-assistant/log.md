# Home Assistant

## Source
- GitHub: https://github.com/home-assistant/core
- Docker Hub: homeassistant/home-assistant

## Description
Open-source home automation platform that puts local control and privacy first. Supports thousands of integrations for smart devices, with automations, dashboards, and voice control.

## Stack
- **homeassistant**: Core HA on host network (for device discovery/mDNS)
- **mosquitto**: Eclipse Mosquitto MQTT broker for device communication
- Bind-mounted mosquitto.conf for broker configuration
- Privileged mode for USB device access
