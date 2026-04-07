import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { router } from "sv-router/vite-plugin";
import path from "path";

export default defineConfig({
    plugins: [tailwindcss(), svelte(), router()],
    build: {
        outDir: path.resolve("../api/internal/spa/dist"),
        emptyOutDir: true,
    },
    server: {
        port: 3000,
    },
    resolve: {
        alias: {
            $lib: path.resolve("./src/lib"),
        },
    },
});
