"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { Bell, Globe, LifeBuoy, Settings as SettingsIcon, ShieldAlert } from "lucide-react";
import { NotificationToggle } from "@/components/NotificationToggle";
import { TelegramLinkCard } from "@/components/TelegramLinkCard";
import { InstallApp } from "@/components/InstallApp";
import { AdminBroadcast } from "@/components/AdminBroadcast";
import { useAuth } from "@/lib/auth";
import { useLang, LANG_LABELS, Lang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

// Контакт поддержки — Telegram. Можно переопределить через NEXT_PUBLIC_SUPPORT_URL.
const SUPPORT_URL = process.env.NEXT_PUBLIC_SUPPORT_URL || "https://t.me/eFoot6all_bot";

function Section({ icon: Icon, title, children }: { icon: typeof Bell; title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-zinc-500">
        <Icon size={14} className="text-yellow-500" /> {title}
      </div>
      {children}
    </div>
  );
}

export default function SettingsPage() {
  const router = useRouter();
  const { user, loading } = useAuth();
  const { t, lang, setLang } = useLang();

  useEffect(() => {
    if (!loading && !user) router.replace("/login");
  }, [loading, user, router]);

  if (loading || !user) return null;

  return (
    <div className="max-w-2xl mx-auto space-y-7">
      <div>
        <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-1">{t("settings.subtitle")}</p>
        <h1 className="font-display text-2xl font-bold text-zinc-100 flex items-center gap-2">
          <SettingsIcon size={22} className="text-yellow-400" /> {t("settings.title")}
        </h1>
      </div>

      {/* Уведомления и связь */}
      <Section icon={Bell} title={t("settings.sectionNotifications")}>
        <NotificationToggle />
        <TelegramLinkCard />
      </Section>

      {/* Приложение + язык */}
      <Section icon={Globe} title={t("settings.sectionApp")}>
        <InstallApp />
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
          <p className="text-sm font-bold text-zinc-100 mb-3">{t("settings.language")}</p>
          <div className="flex gap-2">
            {(["ru", "uz", "tg"] as Lang[]).map((l) => (
              <button
                key={l}
                onClick={() => setLang(l)}
                className={cn(
                  "flex-1 rounded-lg py-2 text-sm font-semibold border transition-colors",
                  lang === l
                    ? "bg-yellow-400 border-yellow-400 text-zinc-950"
                    : "bg-zinc-800 border-zinc-700 text-zinc-300 hover:bg-zinc-700"
                )}
              >
                {LANG_LABELS[l]}
              </button>
            ))}
          </div>
        </div>
      </Section>

      {/* Поддержка */}
      <Section icon={LifeBuoy} title={t("settings.sectionSupport")}>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-[#229ED9]/15 text-[#229ED9]">
              <LifeBuoy size={20} />
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-bold text-zinc-100">{t("settings.support")}</p>
              <p className="text-xs text-zinc-400 mt-0.5">{t("settings.supportDesc")}</p>
            </div>
          </div>
          <a
            href={SUPPORT_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-[#229ED9] hover:bg-[#1a8bbf] py-2.5 text-sm font-bold text-white transition-colors"
          >
            {t("settings.contactSupport")}
          </a>
        </div>
      </Section>

      {/* Админ: рассылка */}
      {(user.is_admin || user.is_super_admin) && (
        <Section icon={ShieldAlert} title={t("settings.sectionAdmin")}>
          <AdminBroadcast />
        </Section>
      )}
    </div>
  );
}
