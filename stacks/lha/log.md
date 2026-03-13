# LHA (Lightweight Home Automation)

## Source
- GitHub: https://github.com/javalikescript/lha

## Description
Lightweight home automation application built in Lua. Enriches existing gateways (Hue, Z-Wave), records historical data, supports Blockly scripts, and provides a web interface.

## Stack
- **lha**: Single Lua-based process on port 8080
- Volume for persistent data (thing records, scripts, config)

## Notes
- Very small footprint (~5MB), suitable for Raspberry Pi
- No official Docker Hub image found; may need custom image
