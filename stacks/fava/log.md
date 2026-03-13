# Fava

- **Source**: https://github.com/beancount/fava
- **Image**: `yegle/fava:latest` (Docker Hub, 811K+ pulls)
- **Status**: created
- **Notes**: Fava is the web interface for Beancount double-entry accounting. There is no official Docker image from the beancount org; `yegle/fava` is the most popular community image. Users need to mount their beancount files into the `/bean` volume and set BEANCOUNT_FILE accordingly.
