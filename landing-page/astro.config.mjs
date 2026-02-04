// @ts-check
import { defineConfig } from "astro/config";
import tailwindcss from "@tailwindcss/vite";
import mdx from "@astrojs/mdx";
import embeds from "astro-embed/integration";

// https://astro.build/config
export default defineConfig({
  integrations: [embeds(), mdx()],
  vite: { plugins: [tailwindcss()] },
});
