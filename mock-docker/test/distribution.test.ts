import { describe, it, expect } from "vitest";
import { generateStack } from "../src/generator.js";
import { parseCompose } from "../src/compose-parser.js";
import { parseStackMockConfig } from "../src/mock-config.js";
import { FixedClock } from "../src/clock.js";
import type { GeneratorInput } from "../src/generator.js";

function makeInput(yaml: string, mockYaml: string | null = null): GeneratorInput {
    return {
        project: "test-project",
        stackDir: "/opt/stacks/test-project",
        composeFilePath: "/opt/stacks/test-project/compose.yaml",
        parsed: parseCompose(yaml),
        mockConfig: parseStackMockConfig(mockYaml),
        clock: new FixedClock(new Date("2025-01-15T00:00:00Z")),
    };
}

describe("update_available label", () => {
    it("does not set label when updateAvailable is not set", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
`);
        const result = generateStack(input);
        const labels = result.containers[0].Config.Labels!;
        expect(labels["com.portge.mock.update_available"]).toBeUndefined();
    });

    it("sets update_available label when mock config has updateAvailable", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
`, `
services:
  web:
    update_available: true
`);
        const result = generateStack(input);
        const labels = result.containers[0].Config.Labels!;
        expect(labels["com.portge.mock.update_available"]).toBe("true");
    });

    it("sets needs_recreation label and alters Config.Image", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:1.27
`, `
services:
  web:
    needs_recreation: true
`);
        const result = generateStack(input);
        const container = result.containers[0];
        const labels = container.Config.Labels!;
        expect(labels["com.portge.mock.needs_recreation"]).toBe("true");
        // Config.Image should be altered (different from compose image ref)
        expect(container.Config.Image).toBe("nginx:1.27-old");
    });

    it("Config.Image is normal when needsRecreation is not set", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:1.27
`);
        const result = generateStack(input);
        expect(result.containers[0].Config.Image).toBe("nginx:1.27");
    });
});

describe("digest alteration for update_available", () => {
    it("normal container has same digest across calls", () => {
        const input = makeInput(`
services:
  web:
    image: nginx:latest
`);
        const result = generateStack(input);
        const img = result.images.find((i) => i.RepoTags.includes("nginx:latest"))!;
        expect(img.RepoDigests.length).toBeGreaterThan(0);
        // Digest should be stable
        const digest = img.RepoDigests[0].split("@")[1];
        expect(digest).toMatch(/^sha256:[0-9a-f]{64}$/);
    });

    it("update_available container stores label on compose image label", () => {
        const input = makeInput(`
services:
  db:
    image: postgres:16
`, `
services:
  db:
    update_available: true
`);
        const result = generateStack(input);
        const labels = result.containers[0].Config.Labels!;
        expect(labels["com.portge.mock.update_available"]).toBe("true");
        expect(labels["com.docker.compose.image"]).toBe("postgres:16");
    });
});
