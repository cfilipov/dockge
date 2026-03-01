<template>
    <div class="shadow-box cheatsheet">
        <div class="cheatsheet-section">
            <h6 class="section-heading">Stack Lifecycle</h6>
            <div class="cmd-list" @click="insertCommand">
                <div class="cmd"><span class="cmd-desc">Start stack</span><code>docker compose up -d</code></div>
                <div class="cmd"><span class="cmd-desc">Stop &amp; remove</span><code>docker compose down</code></div>
                <div class="cmd"><span class="cmd-desc">Restart services</span><code>docker compose restart</code></div>
                <div class="cmd"><span class="cmd-desc">Stop services</span><code>docker compose stop</code></div>
                <div class="cmd"><span class="cmd-desc">Start stopped</span><code>docker compose start</code></div>
                <div class="cmd"><span class="cmd-desc">Build &amp; recreate</span><code>docker compose up -d --build --force-recreate</code></div>
            </div>
        </div>

        <div class="cheatsheet-section">
            <h6 class="section-heading">Logs &amp; Debug</h6>
            <div class="cmd-list" @click="insertCommand">
                <div class="cmd"><span class="cmd-desc">Follow logs</span><code>docker compose logs -f</code></div>
                <div class="cmd"><span class="cmd-desc">List containers</span><code>docker compose ps</code></div>
                <div class="cmd"><span class="cmd-desc">Running processes</span><code>docker compose top</code></div>
                <div class="cmd"><span class="cmd-desc">Shell into service</span><code>docker compose exec &lt;svc&gt; bash</code></div>
            </div>
        </div>

        <div class="cheatsheet-section">
            <h6 class="section-heading">Images</h6>
            <div class="cmd-list" @click="insertCommand">
                <div class="cmd"><span class="cmd-desc">Pull latest images</span><code>docker compose pull</code></div>
                <div class="cmd"><span class="cmd-desc">Rebuild from scratch</span><code>docker compose build --no-cache</code></div>
                <div class="cmd"><span class="cmd-desc">List images</span><code>docker images</code></div>
            </div>
        </div>

        <div class="cheatsheet-section">
            <h6 class="section-heading">Cleanup</h6>
            <div class="cmd-list" @click="insertCommand">
                <div class="cmd"><span class="cmd-desc">Remove unused data</span><code>docker system prune</code></div>
                <div class="cmd"><span class="cmd-desc">Remove unused volumes</span><code>docker volume prune</code></div>
                <div class="cmd"><span class="cmd-desc">Remove all unused images</span><code>docker image prune -a</code></div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useSocket } from "../composables/useSocket";

const { emit } = useSocket();

function insertCommand(event: MouseEvent) {
    const code = (event.target as HTMLElement).closest("code");
    if (!code) return;
    const text = code.textContent || "";
    emit("terminalInput", "console", text);
}
</script>

<style scoped lang="scss">
.cheatsheet {
    padding: 15px;
    font-size: 0.85rem;
}

.cheatsheet-section {
    margin-bottom: 12px;

    &:last-child {
        margin-bottom: 0;
    }
}

.section-heading {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    opacity: 0.6;
    margin-bottom: 4px;
}

.cmd-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.cmd {
    display: flex;
    flex-direction: column;
    line-height: 1.4;

    code {
        font-size: 0.8rem;
        align-self: flex-start;
        cursor: pointer;

        &:hover {
            opacity: 0.7;
        }
    }
}

.cmd-desc {
    opacity: 0.5;
    font-size: 0.75rem;
}
</style>
