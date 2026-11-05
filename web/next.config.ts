import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Emits .next/standalone with a self-contained server.js for build/Dockerfile.web.
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
