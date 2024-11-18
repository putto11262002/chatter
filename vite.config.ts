import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  root: "web",
  publicDir: "../public",
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./web", import.meta.url)),
    },
  },
});
