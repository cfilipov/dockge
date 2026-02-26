import { test, expect } from "../fixtures/auth.fixture";
import { takeLightScreenshot } from "../helpers/light-mode";

// Login tests use a fresh context — no auth, but keep theme: "auto" so
// emulateMedia can toggle light/dark via the useTheme composable.
test.use({
    storageState: {
        cookies: [],
        origins: [{
            origin: `http://localhost:${process.env.E2E_PORT || "5051"}`,
            localStorage: [{ name: "theme", value: "auto" }],
        }],
    },
});

test.describe("Login Page", () => {
    test.beforeEach(async ({ page }) => {
        await page.goto("/");
        await expect(page.getByPlaceholder("Username")).toBeVisible({ timeout: 15000 });
    });

    test("displays login form with username, password, and Login button", async ({ page }) => {
        await expect(page.getByPlaceholder("Username")).toBeVisible();
        await expect(page.getByPlaceholder("Password")).toBeVisible();
        await expect(page.getByRole("button", { name: "Login" })).toBeVisible();
    });

    test("shows error on invalid credentials", async ({ page }) => {
        await page.getByPlaceholder("Username").fill("wronguser");
        await page.getByPlaceholder("Password").fill("wrongpass");
        await page.getByRole("button", { name: "Login" }).click();

        // Error message should appear
        await expect(page.getByText("Incorrect username or password")).toBeVisible({ timeout: 5000 });
    });

    test("logs in successfully with valid credentials", async ({ page }) => {
        await page.getByPlaceholder("Username").fill("admin");
        await page.getByPlaceholder("Password").fill("testpass123");
        await page.getByRole("button", { name: "Login" }).click();

        await expect(page.getByRole("heading", { name: "Stacks" })).toBeVisible({ timeout: 15000 });
    });

    test("screenshot: login form", async ({ page }) => {
        await expect(page).toHaveScreenshot("login-form.png");
        await takeLightScreenshot(page, "login-form-light.png");
    });
});
