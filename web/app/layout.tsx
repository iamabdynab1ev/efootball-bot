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
import { OnboardingPrompt } from "@/components/OnboardingPrompt";
import { UpdatePrompt } from "@/components/UpdatePrompt";
import { AppSignals } from "@/components/AppSignals";
import { AwardCelebration } from "@/components/AwardCelebration";
import { LogoShowcase } from "@/components/LogoShowcase";

const APP_URL = process.env.NEXT_PUBLIC_APP_URL || "https://efootball.uz";

export const metadata: Metadata = {
  metadataBase: new URL(APP_URL),
  title: {
    default: "eFootLeague",
    template: "%s | eFootLeague",
  },
  description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
  manifest: "/manifest.json",
  robots: { index: true, follow: true },
  openGraph: {
    type: "website",
    siteName: "eFootLeague",
    title: "eFootLeague",
    description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
    url: APP_URL,
    locale: "ru_RU",
    // Логотип в превью ссылок (Telegram/WhatsApp/соцсети).
    images: [{ url: "/icon-512.png", width: 512, height: 512, alt: "eFootball Champions Cup" }],
  },
  twitter: {
    card: "summary",
    title: "eFootLeague",
    description: "Онлайн-лиги eFootball 2026: регистрация, таблицы, матчи, рейтинги и плей-офф.",
    images: ["/icon-512.png"],
  },
  appleWebApp: {
    capable: true,
    statusBarStyle: "black-translucent",
    title: "eFootLeague",
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
  // Заполняем экран под чёлку/home-индикатор. Без этого env(safe-area-inset-*)
  // всегда 0, и safe-area отступы шапки/нижней навигации (Navbar, main) мертвы
  // на iPhone с вырезом и в PWA/Telegram Mini App.
  viewportFit: "cover",
  // Виртуальная клавиатура ужимает layout-вьюпорт (а не наезжает поверх),
  // поэтому 100dvh/полноэкранный чат не прячет поле ввода под клавиатурой.
  interactiveWidget: "resizes-content",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" className="dark">
      <head>
        {/* preconnect только на страницах где есть логотипы клубов */}
        <link rel="dns-prefetch" href="https://r2.thesportsdb.com" />
        <link rel="preconnect" href="https://r2.thesportsdb.com" crossOrigin="anonymous" />
        <link rel="icon" href="/favicon.ico" sizes="48x48" />
        <link rel="apple-touch-icon" href="/apple-touch-icon.png?v=2" />
        <link rel="canonical" href={APP_URL} />
      </head>
      {/* Без bg на body: непрозрачный фон body перекрывал бы стадионные слои
          body::before (порядок отрисовки CSS); базовый цвет красит html. */}
      <body className="text-zinc-100 antialiased">
        <Providers>
          <Navbar />
          <AppSignals />
          <AwardCelebration />
          <LogoShowcase />
          <OnboardingPrompt />
          <UpdatePrompt />
          <SidebarOffset>
            <main id="main-content" className="min-h-screen pb-[calc(5rem+env(safe-area-inset-bottom))] lg:pb-0 mt-[calc(3.5rem+env(safe-area-inset-top))] lg:mt-0">
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
