"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Megaphone, Send } from "lucide-react";
import { toast } from "sonner";
import { adminBroadcast } from "@/lib/api";
import { useLang } from "@/lib/i18n";

/**
 * Админ-панель рассылки: текст → push всем подписанным + Telegram всем привязанным.
 */
export function AdminBroadcast() {
  const { t } = useLang();
  const [text, setText] = useState("");

  const mutation = useMutation({
    mutationFn: () => adminBroadcast(text.trim()),
    onSuccess: (res) => {
      toast.success(`${t("settings.broadcastSent")} · push: ${res.pushed} · TG: ${res.telegram}`);
      setText("");
    },
    onError: () => toast.error(t("settings.broadcastError")),
  });

  return (
    <div className="rounded-xl border border-yellow-500/25 bg-gradient-to-br from-yellow-500/8 to-zinc-900 p-4 space-y-3">
      <div className="flex items-center gap-3">
        <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-yellow-400 text-zinc-950">
          <Megaphone size={20} />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-bold text-zinc-100">{t("settings.broadcast")}</p>
          <p className="text-xs text-zinc-400 mt-0.5">{t("settings.broadcastDesc")}</p>
        </div>
      </div>

      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder={t("settings.broadcastPlaceholder")}
        rows={4}
        maxLength={1000}
        className="w-full rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none focus:border-yellow-500 resize-none"
      />
      <div className="flex items-center justify-between">
        <span className="text-[11px] text-zinc-500">{text.length}/1000</span>
        <button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending || text.trim().length === 0}
          className="flex items-center gap-2 rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          <Send size={15} />
          {mutation.isPending ? "..." : t("settings.broadcastSend")}
        </button>
      </div>
    </div>
  );
}
