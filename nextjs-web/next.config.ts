import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Pin the workspace root: a stray lockfile higher up the tree otherwise makes
  // Turbopack infer the wrong root (and watch a huge directory in dev).
  turbopack: {
    root: import.meta.dirname,
  },
};

export default nextConfig;
