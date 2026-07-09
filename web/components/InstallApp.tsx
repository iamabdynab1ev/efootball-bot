"use client";

import { useEffect, useState } from "react";
import { Download, Share, Plus, Smartphone, Globe } from "lucide-react";
import { useLang } from "@/lib/i18n";

// beforeinstallprompt не типизирован в стандартном lib.dom
interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

/**
 * Кнопка установки PWA на главный экран.
 * - Android/Chrome: ловим beforeinstallprompt → системная установка в один тап.
 * - iPhone/Safari: события нет, показываем инструкцию (Поделиться → На экран «Домой»).
 * - Уже установлено: ничего не показываем.
 */
export function InstallApp() {
  const { t } = useLang();
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
  const [installed, setInstalled] = useState(false);
  const [isIOS, setIsIOS] = useState(false);
  const [showIosHint, setShowIosHint] = useState(false);

  useEffect(() => {
    // Уже установлено (запущено в standalone-режиме) → не показываем
    const standalone =
      window.matchMedia("(display-mode: standalone)").matches ||
      // @ts-expect-error — iOS Safari
      window.navigator.standalone === true;
    if (standalone) {
      setInstalled(true);
      return;
    }

    const ua = window.navigator.userAgent.toLowerCase();
    setIsIOS(/iphone|ipad|ipod/.test(ua));

    const onPrompt = (e: Event) => {
      e.preventDefault();
      setDeferred(e as BeforeInstallPromptEvent);
    };
    const onInstalled = () => {
      setInstalled(true);
      setDeferred(null);
    };
    window.addEventListener("beforeinstallprompt", onPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  // Уже установлено → скрываем блок целиком
  if (installed) return null;

  async function handleInstall() {
    if (deferred) {
      await deferred.prompt();
      const choice = await deferred.userChoice;
      if (choice.outcome === "accepted") setInstalled(true);
      setDeferred(null);
    } else if (isIOS) {
      setShowIosHint((v) => !v);
    }
  }

  // На десктопе без поддержки установки и не iOS — не показываем
  if (!deferred && !isIOS) return null;

  return (
    <div className="space-y-3">
      {/* Заголовок секции рендерим здесь, а не в settings — чтобы при
          отсутствии кнопки установки (десктоп/уже установлено) не оставался
          пустой заголовок «Приложение». */}
      <div className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-zinc-500">
        <Globe size={14} className="text-yellow-500" /> {t("settings.sectionApp")}
      </div>
      <div className="rounded-xl border border-yellow-500/25 bg-gradient-to-br from-yellow-500/10 to-zinc-900 p-4">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-400 text-zinc-950">
          <Smartphone size={20} strokeWidth={2.5} />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("profile.installApp")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{t("profile.installDesc")}</p>
        </div>
      </div>

      <button
        onClick={handleInstall}
        className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-yellow-400 py-2.5 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90"
      >
        <Download size={16} />
        {t("profile.installBtn")}
      </button>

      {/* iOS-инструкция */}
      {showIosHint && (
        <div className="mt-3 rounded-lg border border-zinc-700 bg-zinc-950/60 p-3 text-xs text-zinc-300 space-y-2">
          <div className="flex items-center gap-2">
            <Share size={14} className="text-yellow-400 flex-shrink-0" />
            <span>{t("profile.installIosHint")}</span>
          </div>
          <div className="flex items-center gap-2 text-zinc-500">
            <Plus size={14} className="flex-shrink-0" />
            <span>«{t("profile.installApp")}»</span>
          </div>
        </div>
      )}
      </div>
    </div>
  );
}
