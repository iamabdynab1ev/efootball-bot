import { PHASE_DEVELOPMENT_SERVER } from "next/constants.js";

/** @type {import('next').NextConfig} */
const createConfig = (phase) => ({
  output: "export",
  distDir: phase === PHASE_DEVELOPMENT_SERVER ? ".next" : "../cmd/bot/ui",
  trailingSlash: false,
  images: { unoptimized: true },
});

export default createConfig;
