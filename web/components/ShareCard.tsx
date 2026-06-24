"use client";

import { useState } from "react";
import { Share2, Download, Check } from "lucide-react";
import { toast } from "sonner";
import { playerCardUrl } from "@/lib/api";
import { useLang } from "@/lib/i18n";

/**
 * Кнопка «Поделиться карточкой игрока».
 * - Если устройство поддерживает Web Share с файлами (мобилки) — шарит PNG напрямую
 *   (в Telegram/WhatsApp/соцсети).
 * - Иначе — скачивает карточку.
 */
export function ShareCard({ playerId, name }: { playerId: number; name: string }) {
  const { t } = useLang();
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);

  async function share() {
    setBusy(true);
    try {
      const res = await fetch(playerCardUrl(playerId), { cache: "no-store" });
      const blob = await res.blob();
      const file = new File([blob], `efootleague-${name}.png`, { type: "image/png" });

      // Web Share API с файлами (мобильные браузеры)
      const navAny = navigator as Navigator & { canShare?: (d: ShareData) => boolean };
      if (navAny.canShare && navAny.canShare({ files: [file] })) {
        await navigator.share({
          files: [file],
          title: `${name} — eFootLeague`,
          text: `Моя карточка в eFootLeague 🏆`,
        });
      } else {
        // Фолбэк — скачивание
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `efootleague-${name}.png`;
        a.click();
        URL.revokeObjectURL(url);
        toast.success(t("share.downloaded"));
      }
      setDone(true);
      setTimeout(() => setDone(false), 2000);
    } catch {
      toast.error(t("share.error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <button
      onClick={share}
      disabled={busy}
      className="flex items-center justify-center gap-2 rounded-lg bg-yellow-400 px-4 py-2.5 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
    >
      {done ? <Check size={16} /> : <Share2 size={16} />}
      {busy ? t("share.preparing") : t("share.button")}
    </button>
  );
}
