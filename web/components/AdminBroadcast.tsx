"use client";

import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Megaphone, Send, Users, User as UserIcon, Search } from "lucide-react";
import { toast } from "sonner";
import { adminBroadcast, adminNotifyUser, fetchPlayers } from "@/lib/api";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { useLang } from "@/lib/i18n";
import { cn } from "@/lib/utils";

type Mode = "all" | "one";

/**
 * Админ-панель уведомлений: рассылка всем ИЛИ адресно одному игроку.
 * Канал: web push (подписанным) + Telegram (привязанным).
 */
export function AdminBroadcast() {
  const { t } = useLang();
  const [mode, setMode] = useState<Mode>("all");
  const [text, setText] = useState("");
  const [targetId, setTargetId] = useState<number | null>(null);
  const [search, setSearch] = useState("");

  const { data: players = [] } = useQuery({
    queryKey: ["players", 300],
    queryFn: () => fetchPlayers(300),
    enabled: mode === "one",
    staleTime: 60000,
  });

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return players.slice(0, 30);
    return players.filter((p) => p.display_name.toLowerCase().includes(q)).slice(0, 30);
  }, [players, search]);

  const target = players.find((p) => p.id === targetId);

  const mutation = useMutation({
    mutationFn: () =>
      mode === "all" ? adminBroadcast(text.trim()) : adminNotifyUser(targetId!, text.trim()),
    onSuccess: (res: { pushed: number; telegram: number }) => {
      toast.success(
        `${mode === "all" ? t("settings.broadcastSent") : t("settings.notifySent")} · push: ${res.pushed} · TG: ${res.telegram}`
      );
      setText("");
      if (mode === "one") { setTargetId(null); setSearch(""); }
    },
    onError: () => toast.error(t("settings.broadcastError")),
  });

  const canSend = text.trim().length > 0 && (mode === "all" || targetId !== null);

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

      {/* Переключатель: всем / одному */}
      <div className="flex gap-1 rounded-lg border border-zinc-800 bg-zinc-900 p-1">
        {([
          { key: "all" as Mode, label: t("settings.targetAll"), icon: Users },
          { key: "one" as Mode, label: t("settings.targetPlayer"), icon: UserIcon },
        ]).map(({ key, label, icon: Icon }) => (
          <button
            key={key}
            onClick={() => { setMode(key); }}
            className={cn(
              "flex flex-1 items-center justify-center gap-2 rounded-md py-2 text-sm font-semibold transition-colors",
              mode === key ? "bg-yellow-400 text-zinc-950" : "text-zinc-400 hover:text-zinc-200"
            )}
          >
            <Icon size={14} /> {label}
          </button>
        ))}
      </div>

      {/* Выбор игрока (режим «одному») */}
      {mode === "one" && (
        <div className="space-y-2">
          {target ? (
            <div className="flex items-center gap-2 rounded-lg border border-yellow-500/40 bg-yellow-500/5 px-3 py-2">
              <PlayerAvatar displayName={target.display_name} favoriteClub={target.favorite_club} size={28} />
              <span className="flex-1 text-sm font-semibold text-zinc-100 truncate">{target.display_name}</span>
              <button onClick={() => setTargetId(null)} className="text-xs text-zinc-400 hover:text-zinc-200">✕</button>
            </div>
          ) : (
            <>
              <div className="relative">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
                <input
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder={t("settings.searchPlayer")}
                  className="w-full rounded-lg border border-zinc-700 bg-zinc-800 pl-8 pr-3 py-2 text-sm text-zinc-100 placeholder-zinc-600 outline-none focus:border-yellow-500"
                />
              </div>
              <div className="max-h-48 overflow-y-auto rounded-lg border border-zinc-800 divide-y divide-zinc-800/60">
                {filtered.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => { setTargetId(p.id); }}
                    className="flex w-full items-center gap-2 px-3 py-2 hover:bg-zinc-800/50 transition-colors text-left"
                  >
                    <PlayerAvatar displayName={p.display_name} favoriteClub={p.favorite_club} size={26} />
                    <span className="flex-1 text-sm text-zinc-200 truncate">{p.display_name}</span>
                    {p.has_telegram && <span className="text-[10px] text-[#229ED9]">TG</span>}
                  </button>
                ))}
                {filtered.length === 0 && (
                  <p className="px-3 py-3 text-xs text-zinc-500 text-center">—</p>
                )}
              </div>
            </>
          )}
        </div>
      )}

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
          disabled={mutation.isPending || !canSend}
          className="flex items-center gap-2 rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-950 transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          <Send size={15} />
          {mutation.isPending ? "..." : (mode === "all" ? t("settings.broadcastSend") : t("settings.broadcastSend"))}
        </button>
      </div>
    </div>
  );
}
