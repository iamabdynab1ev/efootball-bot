"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { LazyMotion, domAnimation } from "framer-motion";
import { useState } from "react";
import { AuthProvider } from "@/lib/auth";
import { LanguageProvider } from "@/lib/i18n";
import { Toaster } from "sonner";

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () => new QueryClient({
      defaultOptions: {
        queries: { staleTime: 20_000, retry: 1, refetchOnWindowFocus: false },
      },
    }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <LanguageProvider>
          {/* LazyMotion + m.* вместо motion.* — экономит ~15-20KB gzip в общем бандле */}
          <LazyMotion features={domAnimation} strict>
            {children}
          </LazyMotion>
          <Toaster
            position="bottom-right"
            toastOptions={{
              style: {
                background: "hsl(240 10% 9%)",
                border: "1px solid hsl(240 6% 18%)",
                color: "hsl(0 0% 93%)",
                fontSize: "13px",
                borderRadius: "10px",
              },
            }}
          />
        </LanguageProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}
