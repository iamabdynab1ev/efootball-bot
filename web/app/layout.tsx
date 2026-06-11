import type { Metadata, Viewport } from "next";
// Шрифты грузятся из npm (woff2 в бандле) — сборка static export не требует сети.
// Inter — текст/таблицы, Unbounded — display для заголовков и счёта.
// Сабсеты: latin + cyrillic + cyrillic-ext (таджикские Ӣ Ӯ Қ Ғ Ҳ Ҷ).
import "@fontsource-variable/inter/wght.css";
import "@fontsource-variable/unbounded/wght.css";
import "./globals.css";
import { Providers } from "./providers";
import { Navbar } from "@/components/Navbar";
import { SidebarOffset } from "@/components/SidebarOffset";
import { ErrorBoundary } from "@/components/ErrorBoundary";

const APP_URL = process.env.NEXT_PUBLIC_APP_URL || "https://efootball.uz";

export const metadata: Metadata = {
  metadataBase: new URL(APP_URL),
  title: {
    default: "eFootball Web League",
    template: "%s | eFootball Web League",
  },
  description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
  manifest: "/manifest.json",
  robots: { index: true, follow: true },
  openGraph: {
    type: "website",
    siteName: "eFootball Web League",
    title: "eFootball Web League",
    description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
    url: APP_URL,
    locale: "ru_RU",
  },
  twitter: {
    card: "summary",
    title: "eFootball Web League",
    description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
  },
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "eFootball",
  },
  other: {
    "mobile-web-app-capable": "yes",
  },
};

export const viewport: Viewport = {
  themeColor: "#c8f135",
  width: "device-width",
  initialScale: 1,
  maximumScale: 5,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" className="dark">
      <head>
        {/* preconnect только на страницах где есть логотипы клубов */}
        <link rel="dns-prefetch" href="https://crests.football-data.org" />
        <link rel="apple-touch-icon" href="/icon-192.png" />
        <link rel="canonical" href={APP_URL} />
      </head>
      <body className="bg-zinc-950 text-zinc-100 antialiased">
        <Providers>
          <Navbar />
          <SidebarOffset>
            <main id="main-content" className="min-h-screen pb-20 lg:pb-0 mt-14 lg:mt-0">
              <div className="mx-auto max-w-6xl px-4 lg:px-8 py-8">
                <ErrorBoundary>
                  {children}
                </ErrorBoundary>
              </div>
            </main>
          </SidebarOffset>
        </Providers>
      </body>
    </html>
  );
}
