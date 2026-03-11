<script module lang="ts">
	import { defineMeta } from "@storybook/addon-svelte-csf";
	import CodeEditor from "./CodeEditor.svelte";
	import { yamlLang, envLang, jsonLang } from "$lib/editor-langs";

	const { Story } = defineMeta({
		title: "Components/CodeEditor",
		argTypes: {
			readonly: { control: "boolean" },
		},
		args: {
			readonly: false,
		},
	});

	const sampleYaml = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
    restart: unless-stopped

  redis:
    image: redis:7
    command: redis-server --appendonly yes
    volumes:
      - redis-data:/data

volumes:
  redis-data:
`;

	const sampleEnv = `# Application settings
APP_NAME=my-app
APP_PORT=3000
APP_ENV=production

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=admin
DB_PASS=secret123
`;

	const sampleJson = `{
  "name": "my-project",
  "version": "1.0.0",
  "scripts": {
    "dev": "vite dev",
    "build": "vite build"
  },
  "dependencies": {
    "svelte": "^5.0.0"
  }
}
`;

	const longYaml = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./html:/usr/share/nginx/html:ro
      - ./config/nginx/nginx.conf:/etc/nginx/conf.d/default.conf:ro
      - ./config/nginx/ssl/certificates:/etc/nginx/ssl/certificates:ro
    environment:
      - VIRTUAL_HOST=myapp.example.com,www.myapp.example.com,staging.myapp.example.com
      - LETSENCRYPT_HOST=myapp.example.com,www.myapp.example.com,staging.myapp.example.com
      - NGINX_ENTRYPOINT_QUIET_LOGS=1
    labels:
      traefik.http.routers.web.rule: "Host(\`myapp.example.com\`) || Host(\`www.myapp.example.com\`) || Host(\`staging.myapp.example.com\`)"
      traefik.http.routers.web.entrypoints: websecure
      traefik.http.services.web.loadbalancer.server.port: "80"
    restart: unless-stopped
`;
</script>

<Story name="YAML">
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleYaml} extensions={yamlLang} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Env File">
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleEnv} extensions={envLang} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="JSON">
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleJson} extensions={jsonLang} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Long Lines">
	{#snippet template(args)}
		<div class="h-80 max-w-md">
			<CodeEditor value={longYaml} extensions={yamlLang} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Fullscreen Button">
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleYaml} extensions={yamlLang} onfullscreen={() => alert("Fullscreen clicked")} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Readonly" args={{ readonly: true }}>
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleYaml} extensions={yamlLang} {...args} />
		</div>
	{/snippet}
</Story>

<Story name="Playground">
	{#snippet template(args)}
		<div class="h-80">
			<CodeEditor value={sampleYaml} extensions={yamlLang} {...args} />
		</div>
	{/snippet}
</Story>
