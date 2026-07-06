import { PHASE_DEVELOPMENT_SERVER } from "next/constants.js";
import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8088";

// Версия сборки: одно и то же значение зашивается в бандл (NEXT_PUBLIC_BUILD_TS)
// и кладётся в public/version.json. Клиент сравнивает их и предлагает
// «Обновить», когда на сервере уже новая версия.
const BUILD_TS = Date.now().toString();
const __dir = dirname(fileURLToPath(import.meta.url));
try {
  writeFileSync(join(__dir, "public", "version.json"), JSON.stringify({ v: BUILD_TS }));
} catch { /* readonly fs — не критично */ }

// Безопасные заголовки — только базовые, без CSP (CSP добавляет Go-сервер)
const prodSecurityHeaders = [
  { key: "X-Content-Type-Options",    value: "nosniff" },
  { key: "X-Frame-Options",           value: "SAMEORIGIN" },
  { key: "X-XSS-Protection",          value: "1; mode=block" },
  { key: "Referrer-Policy",           value: "strict-origin-when-cross-origin" },
  { key: "Permissions-Policy",        value: "geolocation=(), microphone=(), camera=()" },
];

/** @type {import('next').NextConfig} */
const createConfig = (phase) => {
  const isDev = phase === PHASE_DEVELOPMENT_SERVER;
  return {
    output: isDev ? undefined : "export",
    distDir: isDev ? ".next" : "../cmd/bot/ui",
    trailingSlash: false,
    images: { unoptimized: true },
    env: { NEXT_PUBLIC_BUILD_TS: BUILD_TS },
    experimental: {
      optimizePackageImports: ["lucide-react", "@tanstack/react-query"],
    },
    // В dev режиме — только rewrites (проксируем API на Go-сервер)
    // Заголовки НЕ добавляем в dev, чтобы не ломать HMR и WebSocket
    ...(isDev && {
      async rewrites() {
        return [
          { source: "/auth/:path*", destination: `${API_URL}/auth/:path*` },
          { source: "/api/:path*",  destination: `${API_URL}/api/:path*`  },
        ];
      },
    }),
    // В production заголовки добавляет Go-сервер (router.go)
  };
};

export default createConfig;
