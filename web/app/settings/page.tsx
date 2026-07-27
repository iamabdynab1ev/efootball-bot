"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell, Languages, LifeBuoy, LogOut, Settings as SettingsIcon, ShieldAlert, UserCircle } from "lucide-react";
import { NotificationToggle } from "@/components/NotificationToggle";
import { TelegramLinkCard } from "@/components/TelegramLinkCard";
import { InstallApp } from "@/components/InstallApp";
import { AdminBroadcast } from "@/components/AdminBroadcast";
import { SupportCard } from "@/components/SupportCard";
import { AdminSupportForm } from "@/components/AdminSupportForm";
import { useAuth } from "@/lib/auth";
import { useFeatures } from "@/lib/features";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

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
  const { user, loading, logout } = useAuth();
  const features = useFeatures();
  const { t } = useLang();

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
        {features.telegram && <TelegramLinkCard />}
      </Section>

      {/* Приложение — InstallApp сам рендерит заголовок секции (или ничего,
          если установка недоступна), чтобы не оставался пустой заголовок. */}
      <InstallApp />

      {/* Поддержка */}
      <Section icon={LifeBuoy} title={t("settings.sectionSupport")}>
        <SupportCard />
      </Section>

      {/* Админ: рассылка + контакты поддержки */}
      {(user.is_admin || user.is_super_admin) && (
        <Section icon={ShieldAlert} title={t("settings.sectionAdmin")}>
          <AdminBroadcast />
          <AdminSupportForm />
        </Section>
      )}

      {/* Аккаунт: выход */}
      <Section icon={UserCircle} title={t("settings.sectionAccount")}>
        <button
          onClick={logout}
          className="flex w-full items-center justify-center gap-2 rounded-xl border border-red-500/30 bg-red-500/10 py-3 text-sm font-bold text-red-400 hover:bg-red-500/20 transition-colors"
        >
          <LogOut size={16} /> {t("settings.logout")}
        </button>
      </Section>
    </div>
  );
}
