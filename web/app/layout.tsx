import type { Metadata } from "next";
import "./globals.css";
import { Providers } from "./providers";
import { Navbar } from "@/components/Navbar";
import { SidebarOffset } from "@/components/SidebarOffset";

export const metadata: Metadata = {
  title: "eFootball Web League",
  description: "Веб-приложение для лиг eFootball: заявки, таблицы, матчи, результаты и админка.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" className="dark">
      <body className="bg-zinc-950 text-zinc-100 antialiased">
        <Providers>
          <Navbar />
          {/* Offset main content to the right of sidebar on desktop */}
          <SidebarOffset>
            <main className="min-h-screen pt-0 lg:pt-0 pb-20 lg:pb-0 mt-14 lg:mt-0">
              <div className="mx-auto max-w-6xl px-4 lg:px-8 py-8">
                {children}
              </div>
            </main>
          </SidebarOffset>
        </Providers>
      </body>
    </html>
  );
}
