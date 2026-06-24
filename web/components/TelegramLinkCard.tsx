"use client";

import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Bot, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { generateLinkCode } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useLang } from "@/lib/i18n";

/**
 * Карточка привязки Telegram: показывает статус, либо кнопку для генерации кода
 * и deep-link на бота. Поллит статус до 2 минут после открытия бота.
 */
export function TelegramLinkCard() {
  const { user, refreshUser } = useAuth();
  const { t } = useLang();
  const [linkInfo, setLinkInfo] = useState<{ code: string; deepLink?: string } | null>(null);
  const [waiting, setWaiting] = useState(false);

  const linkMutation = useMutation({
    mutationFn: generateLinkCode,
    onSuccess: (data) => {
      setLinkInfo({ code: data.code, deepLink: data.deep_link });
      setWaiting(true);
    },
    onError: () => toast.error(t("profile.codeError")),
  });

  useEffect(() => {
    if (!waiting) return;
    const interval = setInterval(async () => {
      await refreshUser();
      if (user?.has_telegram) {
        setWaiting(false);
        clearInterval(interval);
        toast.success(t("profile.telegramLinked"));
      }
    }, 3000);
    const timeout = setTimeout(() => { clearInterval(interval); setWaiting(false); }, 120000);
    return () => { clearInterval(interval); clearTimeout(timeout); };
  }, [waiting, refreshUser, user?.has_telegram, t]);

  if (user?.has_telegram) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4">
        <div className="flex items-center gap-3 rounded-lg bg-green-500/10 border border-green-500/20 px-3 py-2.5">
          <ShieldCheck size={18} className="text-green-400 flex-shrink-0" />
          <div>
            <p className="text-sm font-semibold text-green-300">{t("profile.telegramLinked")}</p>
            <p className="text-xs text-zinc-400">
              {user.username ? `@${user.username}` : t("profile.notificationsAvailable")}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900 p-4 space-y-3">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-[#229ED9]/15 text-[#229ED9]">
          <Bot size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("profile.linkTelegramTitle")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{t("profile.linkTelegramDesc")}</p>
        </div>
      </div>

      {linkInfo ? (
        <>
          {linkInfo.deepLink && (
            <a
              href={linkInfo.deepLink}
              target="_blank"
              rel="noopener noreferrer"
              className="flex w-full items-center justify-center gap-2 rounded-lg bg-[#229ED9] hover:bg-[#1a8bbf] text-white text-sm font-semibold py-2.5 transition-colors"
            >
              <Bot size={15} /> {t("profile.openTelegramBot")}
            </a>
          )}
          <div className="rounded-lg bg-zinc-950/60 border border-zinc-800 px-3 py-2 text-center">
            <p className="text-[10px] uppercase tracking-wide text-zinc-500">{t("profile.yourCode")}</p>
            <p className="font-display text-lg font-black tracking-widest text-zinc-100">{linkInfo.code}</p>
            <p className="text-[10px] text-zinc-400 mt-0.5">
              {t("profile.sendCodeHint")} <span className="text-zinc-300">/link {linkInfo.code}</span>
            </p>
          </div>
          {waiting && <p className="text-[11px] text-zinc-400 text-center">{t("profile.waitingTelegramHint")}</p>}
        </>
      ) : (
        <button
          onClick={() => {
            const win = window.open("about:blank", "_blank");
            linkMutation.mutate(undefined, {
              onSuccess: (data) => {
                if (win && data.deep_link) win.location.href = data.deep_link;
                else if (win) win.close();
              },
            });
          }}
          disabled={linkMutation.isPending}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-[#229ED9] hover:bg-[#1a8bbf] text-white text-sm font-bold py-2.5 transition-opacity disabled:opacity-50"
        >
          <Bot size={15} />
          {linkMutation.isPending ? t("profile.generatingCode") : t("profile.openTelegramBot")}
        </button>
      )}
    </div>
  );
}
