"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell, Check, CheckCheck, Swords, Trophy, AlertTriangle, ShieldCheck, UserCheck, UserX, Megaphone, AtSign, MessageSquare } from "lucide-react";
import { useNotifications, type Notif } from "@/lib/notifications";
import { cn } from "@/lib/utils";

const ICONS: Record<string, { Icon: typeof Bell; tone: string }> = {
  "match.result":    { Icon: Swords,      tone: "text-sky-400" },
  "match.confirmed": { Icon: Check,       tone: "text-emerald-400" },
  "match.disputed":  { Icon: AlertTriangle, tone: "text-orange-400" },
  "match.resolved":  { Icon: ShieldCheck, tone: "text-yellow-400" },
  "member.approved": { Icon: UserCheck,   tone: "text-emerald-400" },
  "member.rejected": { Icon: UserX,       tone: "text-red-400" },
  "tournament":      { Icon: Trophy,      tone: "text-yellow-400" },
  "mention":         { Icon: AtSign,      tone: "text-sky-400" },
  "direct":          { Icon: MessageSquare, tone: "text-yellow-400" },
  "system":          { Icon: Megaphone,   tone: "text-purple-400" },
};

function iconFor(type: string) {
  return ICONS[type] ?? { Icon: Bell, tone: "text-zinc-400" };
}

function fmtTime(iso: string): string {
  const d = new Date(iso);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return "только что";
  if (diff < 3600) return `${Math.floor(diff / 60)} мин`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч`;
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" });
}

interface Props {
  /** Сторона раскрытия панели относительно кнопки. */
  align?: "left" | "right";
}

export function NotificationBell({ align = "right" }: Props) {
  const { notifications, unread, markRead, markAllRead, loadMore } = useNotifications();
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const ref = useRef<HTMLDivElement>(null);

  // Клик вне панели — закрываем.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  const onItem = (n: Notif) => {
    markRead(n.id);
    if (n.link) router.push(n.link);
    setOpen(false);
  };

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label="Уведомления"
        className="relative rounded-md p-2 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors"
      >
        <Bell size={16} />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </button>

      {open && (
        <div
          className={cn(
            "z-50 rounded-xl border border-zinc-800 bg-zinc-900 shadow-2xl shadow-black/50 overflow-hidden",
            // Мобайл: кнопка стоит в середине шапки, поэтому панель не привязываем
            // к кнопке (иначе уезжает за левый край и обрезается overflow-x:hidden),
            // а пиним под топ-баром на всю ширину минус поля.
            "fixed inset-x-3 top-[calc(3.5rem+env(safe-area-inset-top))] mt-1 w-auto",
            // Десктоп/планшет: обычный dropdown, привязанный к кнопке.
            "sm:absolute sm:inset-x-auto sm:top-full sm:mt-2 sm:w-[360px]",
            align === "right" ? "sm:right-0" : "sm:left-0"
          )}
        >
          <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-2.5">
            <span className="text-xs font-bold uppercase tracking-widest text-zinc-400">Уведомления</span>
            {unread > 0 && (
              <button
                onClick={markAllRead}
                className="flex items-center gap-1 text-[11px] text-zinc-400 hover:text-yellow-400 transition-colors"
              >
                <CheckCheck size={13} /> Прочитать все
              </button>
            )}
          </div>

          <div className="max-h-[60vh] overflow-y-auto overscroll-contain">
            {notifications.length === 0 ? (
              <div className="px-4 py-10 text-center text-sm text-zinc-600">
                <Bell size={20} className="mx-auto mb-2 opacity-40" />
                Пока нет уведомлений
              </div>
            ) : (
              notifications.map((n) => {
                const { Icon, tone } = iconFor(n.type);
                return (
                  <button
                    key={n.id}
                    onClick={() => onItem(n)}
                    className={cn(
                      "flex w-full items-start gap-3 border-b border-zinc-800/40 px-4 py-3 text-left last:border-0 hover:bg-zinc-800/40 transition-colors",
                      !n.read && "bg-yellow-500/[0.04]"
                    )}
                  >
                    <Icon size={17} className={cn("mt-0.5 flex-shrink-0", tone)} />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className={cn("text-sm truncate", n.read ? "text-zinc-300 font-medium" : "text-zinc-100 font-semibold")}>{n.title}</p>
                        {!n.read && <span className="h-1.5 w-1.5 flex-shrink-0 rounded-full bg-yellow-400" />}
                      </div>
                      {n.body && <p className="mt-0.5 text-xs text-zinc-500 line-clamp-2">{n.body}</p>}
                      <p className="mt-1 text-[10px] uppercase tracking-wide text-zinc-600">{fmtTime(n.created_at)}</p>
                    </div>
                  </button>
                );
              })
            )}
          </div>

          {notifications.length >= 30 && (
            <button
              onClick={loadMore}
              className="w-full border-t border-zinc-800 py-2.5 text-xs text-zinc-400 hover:bg-zinc-800/40 hover:text-zinc-200 transition-colors"
            >
              Показать ещё
            </button>
          )}
        </div>
      )}
    </div>
  );
}
