# Vanilla Cookbook

- **Source**: https://github.com/jt196/vanilla-cookbook
- **Description**: Self-hosted recipe manager with smart ingredient parsing, unit conversion, and AI-assisted features
- **Architecture**: SvelteKit with Prisma/SQLite, single container
- **Images**: jt196/vanilla-cookbook
- **Compose reference**: Directly from upstream docker-compose.yml.template
- **Notes**: Features include US volumetric to metric weight conversion, recipe scraping via bookmarklet or URL, nutrition parsing, recipe scaling, and optional LLM integration (OpenAI, Anthropic, Google, Ollama). Bind-mounts db/ and uploads/ for persistent data.
