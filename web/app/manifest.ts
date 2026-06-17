import type { MetadataRoute } from "next";

export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "eFootball League",
    short_name: "eFL",
    description: "Лиги eFootball: таблицы, матчи, рейтинги",
    start_url: "/",
    display: "standalone",
    background_color: "#09090b",
    theme_color: "#eab308",
    orientation: "portrait",
    icons: [
      { src: "/icon.svg", sizes: "any", type: "image/svg+xml", purpose: "any maskable" },
      { src: "/icon-192.png", sizes: "192x192", type: "image/png", purpose: "any maskable" },
      { src: "/icon-512.png", sizes: "512x512", type: "image/png", purpose: "any maskable" },
    ],
  };
}
