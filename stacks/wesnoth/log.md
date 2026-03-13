# The Battle for Wesnoth

## Source
https://github.com/wesnoth/wesnoth

## Description
The Battle for Wesnoth is a turn-based tactical strategy game with a high fantasy theme, featuring singleplayer and online/hotseat multiplayer. This stack runs the dedicated multiplayer server (wesnothd).

## Stack Details
- **server**: Wesnoth multiplayer server (wesnoth/wesnoth:steamrt-sniper) on port 15000

## Configuration
- `wesnothd.cfg`: Server configuration (connections, MOTD, room policy, banned names)
- Default port 15000 (standard Wesnoth multiplayer port)

## Notes
- Using the `steamrt-sniper` tag from the official wesnoth/wesnoth Docker Hub image
- The image is primarily for CI/build purposes but includes the wesnothd server binary
- Server data persisted in named volume
