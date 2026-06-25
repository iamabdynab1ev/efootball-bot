"use client";

import { useEffect, useRef, useState } from "react";
import { Share2 } from "lucide-react";
import { toast } from "sonner";
import { renderCard, canvasToBlob, CardInput } from "@/lib/cardCanvas";
import { useLang } from "@/lib/i18n";

/**
 * Карточка игрока, нарисованная в браузере (canvas) — с настоящим логотипом
 * клуба. Ноль нагрузки на сервер. Кнопка «Поделиться» — только для своей.
 */
export function PlayerCard({ data, canShare }: { data: CardInput; canShare?: boolean }) {
  const { t } = useLang();
  const ref = useRef<HTMLCanvasElement>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const c = ref.current;
    if (!c) return;
    let cancelled = false;
    (async () => {
      try {
        await (document as Document & { fonts?: FontFaceSet }).fonts?.ready;
      } catch { /* ignore */ }
      if (!cancelled) renderCard(c, data);
    })();
    return () => { cancelled = true; };
  }, [data]);

  async function share() {
    const c = ref.current;
    if (!c) return;
    setBusy(true);
    try {
      const blob = await canvasToBlob(c);
      if (!blob) throw new Error("no blob");
      const file = new File([blob], `efootleague-${data.name}.png`, { type: "image/png" });
      const navAny = navigator as Navigator & { canShare?: (d: ShareData) => boolean };
      if (navAny.canShare && navAny.canShare({ files: [file] })) {
        await navigator.share({ files: [file], title: `${data.name} — eFootLeague`, text: "🏆 eFootLeague" });
      } else {
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = file.name;
        a.click();
        URL.revokeObjectURL(url);
        toast.success(t("share.downloaded"));
      }
    } catch {
      toast.error(t("share.error"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-3">
      <canvas
        ref={ref}
        className="w-full rounded-lg border border-zinc-800"
        style={{ aspectRatio: "600 / 350" }}
      />
      {canShare && (
        <button
          onClick={share}
          disabled={busy}
          className="flex w-full items-center justify-center gap-2 rounded-lg bg-yellow-400 py-2.5 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          <Share2 size={16} />
          {busy ? t("share.preparing") : t("share.button")}
        </button>
      )}
    </div>
  );
}
