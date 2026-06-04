import { PHASE_DEVELOPMENT_SERVER } from "next/constants.js";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8088";

/** @type {import('next').NextConfig} */
const createConfig = (phase) => {
  const isDev = phase === PHASE_DEVELOPMENT_SERVER;
  return {
    output: isDev ? undefined : "export",
    distDir: isDev ? ".next" : "../cmd/bot/ui",
    trailingSlash: false,
    images: { unoptimized: true },
    ...(isDev && {
      async rewrites() {
        return [
          { source: "/auth/:path*", destination: `${API_URL}/auth/:path*` },
          { source: "/api/:path*",  destination: `${API_URL}/api/:path*`  },
        ];
      },
    }),
  };
};

export default createConfig;
