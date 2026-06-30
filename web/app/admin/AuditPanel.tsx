"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { Activity, RefreshCw } from "lucide-react";
import { api } from "@/lib/api";
import { useSSE } from "@/lib/sse";
import { cn } from "@/lib/utils";

interface AuditEntry {
  id: number;
  actor_id?: number;
  actor_name?: string;
  action: string;
  entity_type?: string;
  entity_id?: number;
  league_id?: number;
  target_id?: number;
  target_name?: string;
  metadata?: Record<string, any>;
  ip?: string;
  created_at: string;
}

// Словарь действий: подпись + цветовая категория. Совпадает с константами
// аудита на бэкенде (internal/models/audit.go).
const ACTIONS: Record<string, { label: string; tone: string }> = {
  "login":               { label: "Вход",                  tone: "text-zinc-400" },
  "league.join":         { label: "Заявка в лигу",         tone: "text-blue-400" },
  "match.result":        { label: "Ввод счёта",            tone: "text-sky-400" },
  "match.confirm":       { label: "Подтверждение матча",   tone: "text-emerald-400" },
  "match.dispute":       { label: "Спор по матчу",         tone: "text-orange-400" },
  "match.admin_resolve": { label: "Админ: решение по счёту", tone: "text-yellow-400" },
  "league.create":       { label: "Создание лиги",         tone: "text-emerald-400" },
  "league.update":       { label: "Изменение лиги",        tone: "text-blue-400" },
  "league.delete":       { label: "Удаление лиги",         tone: "text-red-400" },
  "league.draw":         { label: "Жеребьёвка",            tone: "text-purple-400" },
  "member.approve":      { label: "Одобрение участника",   tone: "text-emerald-400" },
  "member.reject":       { label: "Отклонение участника",  tone: "text-red-400" },
  "user.delete":         { label: "Удаление пользователя", tone: "text-red-400" },
  "user.ban":            { label: "Бан пользователя",      tone: "text-red-400" },
  "admin.broadcast":     { label: "Рассылка",              tone: "text-purple-400" },
};

function actionMeta(a: string) {
  return ACTIONS[a] ?? { label: a, tone: "text-zinc-400" };
}

function fmtTime(iso: string): string {
  const d = new Date(iso);
  const now = Date.now();
  const diff = (now - d.getTime()) / 1000;
  if (diff < 60) return "только что";
  if (diff < 3600) return `${Math.floor(diff / 60)} мин назад`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч назад`;
  return d.toLocaleString("ru-RU", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function actor(e: AuditEntry): string {
  if (e.actor_name) return e.actor_name;
  if (e.actor_id) return `#${e.actor_id}`;
  return "система";
}

function target(e: AuditEntry): string {
  if (e.target_name) return e.target_name;
  if (e.target_id) return `#${e.target_id}`;
  return "";
}

function metaText(e: AuditEntry): string {
  if (!e.metadata) return "";
  const m = e.metadata;
  if (m.home_goals !== undefined && m.away_goals !== undefined) {
    return `${m.home_goals}:${m.away_goals}${m.note ? ` · ${m.note}` : ""}`;
  }
  if (m.name) return String(m.name);
  if (m.method) return String(m.method);
  const keys = Object.keys(m);
  return keys.length ? keys.map((k) => `${k}=${m[k]}`).join(" · ") : "";
}

const PAGE = 50;

export function AuditPanel() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [action, setAction] = useState("");
  const [actorId, setActorId] = useState("");
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [live, setLive] = useState(true);

  // Текущий фильтр в ref — чтобы live-обработчик SSE видел актуальные значения
  // без переподписки на каждый ввод.
  const filterRef = useRef({ action, actorId });
  filterRef.current = { action, actorId };

  const matchesFilter = useCallback((e: AuditEntry) => {
    const f = filterRef.current;
    if (f.action && e.action !== f.action) return false;
    if (f.actorId && String(e.actor_id ?? "") !== f.actorId.trim()) return false;
    return true;
  }, []);

  const load = useCallback(async (before?: number) => {
    setLoading(true);
    try {
      const params: Record<string, any> = { limit: PAGE };
      if (action) params.action = action;
      if (actorId.trim()) params.actor_id = actorId.trim();
      if (before) params.before = before;
      const r = await api.get("/api/admin/audit", { params });
      const list: AuditEntry[] = r.data.entries ?? [];
      setHasMore(list.length === PAGE);
      setEntries((prev) => (before ? [...prev, ...list] : list));
    } finally {
      setLoading(false);
    }
  }, [action, actorId]);

  // Перезагрузка при смене фильтров (load мемоизирован по action/actorId).
  useEffect(() => { load(); }, [load]);

  // Live-лента: новые события приходят админам через SSE, добавляем сверху.
  useSSE("audit", (e: AuditEntry) => {
    if (!live || !e || typeof e.id !== "number") return;
    if (!matchesFilter(e)) return;
    setEntries((prev) => (prev.some((x) => x.id === e.id) ? prev : [e, ...prev]));
  });

  return (
    <div className="space-y-3">
      {/* Панель фильтров */}
      <div className="flex flex-wrap items-center gap-2">
        <select
          value={action}
          onChange={(e) => setAction(e.target.value)}
          className="rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 focus:border-yellow-400 focus:outline-none"
        >
          <option value="">Все действия</option>
          {Object.entries(ACTIONS).map(([k, v]) => (
            <option key={k} value={k}>{v.label}</option>
          ))}
        </select>
        <input
          value={actorId}
          onChange={(e) => setActorId(e.target.value.replace(/[^0-9]/g, ""))}
          placeholder="ID пользователя"
          inputMode="numeric"
          className="w-40 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-200 placeholder-zinc-600 focus:border-yellow-400 focus:outline-none"
        />
        <button
          onClick={() => load()}
          className="flex items-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-900 px-3 py-2 text-sm text-zinc-300 hover:text-white"
        >
          <RefreshCw size={14} className={loading ? "animate-spin" : ""} /> Обновить
        </button>
        <label className="flex items-center gap-1.5 text-xs text-zinc-400 ml-auto cursor-pointer select-none">
          <span className={cn("flex h-2 w-2 rounded-full", live ? "bg-emerald-400 animate-pulse" : "bg-zinc-600")} />
          <input type="checkbox" checked={live} onChange={(e) => setLive(e.target.checked)} className="accent-yellow-400" />
          Live
        </label>
      </div>

      {/* Таблица */}
      <div className="rounded-xl border border-zinc-800 bg-zinc-900 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="border-b border-zinc-800 text-[10px] font-bold uppercase tracking-widest text-zinc-600">
                <th className="px-3 py-2.5 text-left">Время</th>
                <th className="px-3 py-2.5 text-left">Кто</th>
                <th className="px-3 py-2.5 text-left">Действие</th>
                <th className="px-3 py-2.5 text-left">Объект / цель</th>
                <th className="px-3 py-2.5 text-left hidden sm:table-cell">Детали</th>
                <th className="px-3 py-2.5 text-left hidden md:table-cell">IP</th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 && !loading && (
                <tr><td colSpan={6} className="px-3 py-10 text-center text-zinc-600">
                  <Activity size={20} className="mx-auto mb-2 opacity-40" />
                  Событий нет
                </td></tr>
              )}
              {entries.map((e) => {
                const meta = actionMeta(e.action);
                const tgt = target(e);
                return (
                  <tr key={e.id} className="border-b border-zinc-800/40 last:border-0 hover:bg-zinc-800/30">
                    <td className="px-3 py-2.5 whitespace-nowrap text-zinc-500" title={new Date(e.created_at).toLocaleString("ru-RU")}>
                      {fmtTime(e.created_at)}
                    </td>
                    <td className="px-3 py-2.5 whitespace-nowrap font-medium text-zinc-200">{actor(e)}</td>
                    <td className={cn("px-3 py-2.5 whitespace-nowrap font-semibold", meta.tone)}>{meta.label}</td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-zinc-400">
                      {e.entity_type ? <span className="text-zinc-500">{e.entity_type}{e.entity_id ? ` #${e.entity_id}` : ""}</span> : null}
                      {tgt ? <span className="text-zinc-300">{e.entity_type ? " → " : ""}{tgt}</span> : null}
                    </td>
                    <td className="px-3 py-2.5 text-zinc-400 hidden sm:table-cell max-w-[260px] truncate" title={metaText(e)}>{metaText(e)}</td>
                    <td className="px-3 py-2.5 whitespace-nowrap text-zinc-600 hidden md:table-cell font-mono text-xs">{e.ip || "—"}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {hasMore && entries.length > 0 && (
        <div className="text-center">
          <button
            onClick={() => load(entries[entries.length - 1].id)}
            disabled={loading}
            className="rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-2 text-sm text-zinc-300 hover:text-white disabled:opacity-50"
          >
            {loading ? "Загрузка…" : "Показать ещё"}
          </button>
        </div>
      )}
    </div>
  );
}
