"use client";

import { useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { BellRing, Download, Plus, Send, Share, Smartphone, X } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import { useFeatures } from "@/lib/features";
import { useLang } from "@/lib/i18n";
import { enablePush, isPushEnabled, permissionDenied, pushSupport } from "@/lib/push";

// Единое центр-окно онбординга: пришедшему по ссылке (и любому не установившему
// приложение / без уведомлений) в один тап предлагаем установить PWA и включить
// пуши. Заменяет прежнюю нижнюю карточку EnablePushPrompt, сохраняя её логику:
// определение возможностей устройства, фолбэк на Telegram и откладывание.

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

const SNOOZE_KEY = "onboarding_snooze_until";
const SNOOZE_MS = 24 * 60 * 60 * 1000; // 24 часа — модалка навязчивее карточки

export function OnboardingPrompt() {
  const { user } = useAuth();
  const features = useFeatures();
  const { t } = useLang();
  const pathname = usePathname();

  const [show, setShow] = useState(false);
  const [deferred, setDeferred] = useState<BeforeInstallPromptEvent | null>(null);
  const [isIOS, setIsIOS] = useState(false);
  const [standalone, setStandalone] = useState(true); // считаем установленным, пока не проверили
  const [installedNow, setInstalledNow] = useState(false);
  const [iosHint, setIosHint] = useState(false);
  const [pushDone, setPushDone] = useState(false);
  const [pushBusy, setPushBusy] = useState(false);

  // Окружение + перехват системного beforeinstallprompt (Android/Chrome).
  useEffect(() => {
    const sa =
      window.matchMedia("(display-mode: standalone)").matches ||
      // @ts-expect-error — iOS Safari
      window.navigator.standalone === true;
    setStandalone(sa);
    setIsIOS(/iphone|ipad|ipod/.test(navigator.userAgent.toLowerCase()));

    const onPrompt = (e: Event) => { e.preventDefault(); setDeferred(e as BeforeInstallPromptEvent); };
    const onInstalled = () => { setInstalledNow(true); setDeferred(null); };
    window.addEventListener("beforeinstallprompt", onPrompt);
    window.addEventListener("appinstalled", onInstalled);
    return () => {
      window.removeEventListener("beforeinstallprompt", onPrompt);
      window.removeEventListener("appinstalled", onInstalled);
    };
  }, []);

  const support = typeof window !== "undefined" ? pushSupport() : "unsupported";
  // Что предложить: установка (не в standalone, есть системный промпт или iOS);
  // пуши (залогинен, поддерживаются, не включены, не запрещены); фолбэк на TG
  // (залогинен, пуш недоступен/запрещён, TG не привязан, канал включён).
  const installNeeded = !standalone && !installedNow && (!!deferred || isIOS);
  const pushNeeded = !!user && support === "ok" && !pushDone && !permissionDenied();
  const tgNeeded = !!user && (support !== "ok" || permissionDenied()) && !user.has_telegram && features.telegram;

  useEffect(() => {
    // Не мешаем кинематографичному интро и не лезем в установленное приложение.
    if (standalone || pathname === "/story") { setShow(false); return; }
    if (Date.now() < Number(localStorage.getItem(SNOOZE_KEY) || 0)) return;
    if (!installNeeded && !pushNeeded && !tgNeeded) return;
    let on = true;
    // Небольшая задержка — не бросаем окно в лицо сразу на входе.
    const timer = setTimeout(async () => {
      if (pushNeeded && (await isPushEnabled())) { if (on) setPushDone(true); }
      if (on) setShow(true);
    }, 1800);
    return () => { on = false; clearTimeout(timer); };
  }, [standalone, pathname, installNeeded, pushNeeded, tgNeeded]);

  if (!show) return null;

  const showInstall = installNeeded;
  const showPush = pushNeeded;
  const showTg = tgNeeded && !showPush;
  if (!showInstall && !showPush && !showTg) return null;

  const dismiss = () => {
    localStorage.setItem(SNOOZE_KEY, String(Date.now() + SNOOZE_MS));
    setShow(false);
  };

  const doInstall = async () => {
    if (deferred) {
      await deferred.prompt();
      const choice = await deferred.userChoice;
      setDeferred(null);
      if (choice.outcome === "accepted") {
        setInstalledNow(true);
        if (!showPush && !showTg) setShow(false);
      }
    } else if (isIOS) {
      setIosHint((v) => !v);
    }
  };

  const doPush = async () => {
    setPushBusy(true);
    const res = await enablePush();
    setPushBusy(false);
    if (res === true) {
      setPushDone(true);
      toast.success(t("pushPrompt.enabled"));
      if (!showInstall) { localStorage.removeItem(SNOOZE_KEY); setShow(false); }
    } else if (res === "denied") {
      toast.error(t("pushPrompt.denied"));
    } else {
      toast.error(t("pushPrompt.fail"));
    }
  };

  return (
    <div
      className="fixed inset-0 z-[70] flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm"
      style={{ paddingTop: "max(1rem, env(safe-area-inset-top))", paddingBottom: "max(1rem, env(safe-area-inset-bottom))" }}
      onClick={dismiss}
      role="dialog"
      aria-modal="true"
      aria-label={t("pushPrompt.onbTitle")}
    >
      <div
        className="w-full max-w-sm rounded-2xl card-premium p-5 shadow-2xl shadow-black/50"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="font-display text-lg font-black text-zinc-100">{t("pushPrompt.onbTitle")}</p>
            <p className="mt-0.5 text-xs text-zinc-400">{t("pushPrompt.onbSubtitle")}</p>
          </div>
          <button onClick={dismiss} aria-label={t("pushPrompt.close")} className="-mr-1 -mt-1 flex-shrink-0 rounded-lg p-1.5 text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-300">
            <X size={18} />
          </button>
        </div>

        <div className="mt-4 space-y-2.5">
          {/* Установка приложения */}
          {showInstall && (
            <div className="rounded-xl border border-yellow-500/25 bg-yellow-500/[0.06] p-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-400 text-zinc-950">
                  <Smartphone size={19} strokeWidth={2.5} />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-bold text-zinc-100">{t("profile.installApp")}</p>
                  <p className="mt-0.5 text-[11px] leading-snug text-zinc-400">{t("profile.installDesc")}</p>
                </div>
              </div>
              <button
                onClick={doInstall}
                className="mt-2.5 flex min-h-[42px] w-full items-center justify-center gap-2 rounded-lg bg-yellow-400 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 active:scale-[0.99]"
              >
                <Download size={16} /> {t("profile.installBtn")}
              </button>
              {iosHint && (
                <div className="mt-2.5 space-y-2 rounded-lg border border-zinc-700 bg-zinc-950/60 p-3 text-xs text-zinc-300">
                  <div className="flex items-center gap-2">
                    <Share size={14} className="flex-shrink-0 text-yellow-400" />
                    <span>{t("profile.installIosHint")}</span>
                  </div>
                  <div className="flex items-center gap-2 text-zinc-500">
                    <Plus size={14} className="flex-shrink-0" />
                    <span>«{t("profile.installApp")}»</span>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Включить уведомления */}
          {showPush && (
            <div className="rounded-xl border border-zinc-700 bg-zinc-800/40 p-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-400/15 text-yellow-400">
                  <BellRing size={19} />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-bold text-zinc-100">{t("pushPrompt.title")}</p>
                  <p className="mt-0.5 text-[11px] leading-snug text-zinc-400">{t("pushPrompt.desc")}</p>
                </div>
              </div>
              <button
                onClick={doPush}
                disabled={pushBusy}
                className="mt-2.5 flex min-h-[42px] w-full items-center justify-center gap-2 rounded-lg border border-yellow-400/50 text-sm font-bold text-yellow-400 transition-colors hover:bg-yellow-400/10 disabled:opacity-50"
              >
                <BellRing size={16} /> {pushBusy ? t("pushPrompt.enabling") : t("pushPrompt.enable")}
              </button>
            </div>
          )}

          {/* Пуш недоступен → предлагаем Telegram */}
          {showTg && (
            <div className="rounded-xl border border-sky-500/25 bg-sky-500/[0.06] p-3">
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-sky-400/15 text-sky-400">
                  <Send size={19} />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-bold text-zinc-100">{t("pushPrompt.titleTg")}</p>
                  <p className="mt-0.5 text-[11px] leading-snug text-zinc-400">{t("pushPrompt.descTg")}</p>
                </div>
              </div>
              <a
                href="/settings"
                onClick={() => setShow(false)}
                className="mt-2.5 flex min-h-[42px] w-full items-center justify-center gap-2 rounded-lg bg-sky-500 text-sm font-bold text-white transition-opacity hover:opacity-90 active:scale-[0.99]"
              >
                <Send size={16} /> {t("pushPrompt.linkTg")}
              </a>
            </div>
          )}
        </div>

        <button
          onClick={dismiss}
          className="mt-4 w-full rounded-lg py-2 text-sm font-semibold text-zinc-400 transition-colors hover:bg-zinc-800/60 hover:text-zinc-200"
        >
          {t("pushPrompt.later")}
        </button>
      </div>
    </div>
  );
}
