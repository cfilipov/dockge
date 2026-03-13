# evcc

## Source
- GitHub: https://github.com/evcc-io/evcc
- Docker Hub: evcc/evcc

## Description
EV Charge Controller with PV integration. Manages solar surplus charging for electric vehicles with support for many chargers, meters, and vehicles.

## Stack
- **evcc**: Main application on port 7070 (host networking for device access)
- Bind-mounted evcc.yaml configuration
- Volume for persistent data
