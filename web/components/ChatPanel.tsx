"use client";

import { useEffect, useMemo, useState } from "react";
import { MessageSquare, Archive } from "lucide-react";
import { fetchRoomReads, fetchRoomReactions, useChatMembers, useChatRoom, useChatRooms, type ReactionAgg } from "@/lib/chat";
import { ChatThread } from "@/components/ChatThread";
import { cn } from "@/lib/utils";

interface Props {
  leagueId: number;
  currentUserId?: number;
  isAdmin?: boolean;
  // "card" — ограниченная высота с рамкой (во вкладке); "full" — на всю высоту
  // родителя без рамки (отдельная страница чата).
  variant?: "card" | "full";
}

export function ChatPanel({ leagueId, currentUserId, isAdmin = false, variant = "card" }: Props) {
  const { rooms, loading: roomsLoading } = useChatRooms(leagueId);
  const [roomId, setRoomId] = useState<number | null>(null);
  const room = useMemo(() => rooms.find((r) => r.id === roomId) ?? null, [rooms, roomId]);

  useEffect(() => {
    if (roomId == null && rooms.length > 0) setRoomId(rooms[0].id);
  }, [rooms, roomId]);

  const { messages, loading: msgsLoading, hasMore, send, sendVoice, sendPhoto, loadOlder } = useChatRoom(roomId);
  const members = useChatMembers(roomId); // @упоминания — участники ИМЕННО этой комнаты

  // Начальный прогресс прочтения + реакции комнаты.
  const [initialReads, setInitialReads] = useState<Record<number, number>>({});
  const [initialReactions, setInitialReactions] = useState<ReactionAgg[]>([]);
  useEffect(() => {
    if (roomId == null) { setInitialReads({}); setInitialReactions([]); return; }
    let on = true;
    fetchRoomReads(roomId).then((m) => { if (on) setInitialReads(m); }).catch(() => { /* без галочек */ });
    fetchRoomReactions(roomId).then((rx) => { if (on) setInitialReactions(rx); }).catch(() => { /* без реакций */ });
    return () => { on = false; };
  }, [roomId]);

  if (roomsLoading) {
    // Skeleton вместо текстовой заглушки — та же высота, что у готового чата.
    return (
      <div className={cn(
        "flex flex-col justify-end gap-2.5 bg-zinc-900 p-4",
        variant === "full" ? "h-full" : "h-[75dvh] sm:h-[600px] rounded-xl border border-zinc-800",
      )} aria-hidden>
        {[
          { own: false, w: "w-44", h: "h-12" },
          { own: true, w: "w-56", h: "h-12" },
          { own: false, w: "w-36", h: "h-10" },
          { own: true, w: "w-64", h: "h-14" },
        ].map((s, i) => (
          <div key={i} className={cn("flex", s.own && "justify-end")}>
            <div className={cn("skeleton rounded-2xl", s.own ? "rounded-br-md" : "rounded-bl-md", s.w, s.h)} />
          </div>
        ))}
      </div>
    );
  }
  if (rooms.length === 0) {
    return (
      <div className="rounded-xl card-premium px-4 py-10 text-center">
        <MessageSquare size={22} className="mx-auto mb-2 text-zinc-600" />
        <p className="text-sm text-zinc-500">Чат станет доступен после жеребьёвки и распределения по группам.</p>
      </div>
    );
  }

  return (
    <div className={cn(
      "flex flex-col bg-zinc-900 overflow-hidden",
      variant === "full" ? "h-full" : "h-[75dvh] sm:h-[600px] rounded-xl border border-zinc-800",
    )}>
      {/* Комнаты */}
      {rooms.length > 1 && (
        <div className="flex gap-1.5 border-b border-white/5 bg-zinc-950/40 px-2 py-2 overflow-x-auto scrollbar-none flex-shrink-0">
          {rooms.map((r) => (
            <button
              key={r.id}
              onClick={() => setRoomId(r.id)}
              className={cn(
                "flex flex-shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3.5 py-1.5 text-xs font-semibold transition-all",
                r.id === roomId
                  ? "volt-grad text-zinc-950 shadow-[0_2px_10px_var(--volt-glow-soft)]"
                  : "bg-white/[0.04] text-zinc-400 hover:bg-white/[0.08] hover:text-zinc-100"
              )}
            >
              {r.title}
              {r.archived && <Archive size={11} className="opacity-70" />}
              {r.id !== roomId && (r.unread ?? 0) > 0 && (
                <span className="flex h-4 min-w-[16px] items-center justify-center rounded-full bg-yellow-400 px-1 text-[10px] font-bold text-zinc-900">
                  {(r.unread ?? 0) > 99 ? "99+" : r.unread}
                </span>
              )}
            </button>
          ))}
        </div>
      )}

      <div className="flex-1 min-h-0">
        <ChatThread
          key={roomId ?? 0}
          messages={messages}
          loading={msgsLoading}
          hasMore={hasMore}
          loadOlder={loadOlder}
          send={send}
          sendVoice={sendVoice}
          sendPhoto={sendPhoto}
          currentUserId={currentUserId}
          isAdmin={isAdmin}
          archived={!!room?.archived}
          archivedNote="Турнир завершён — чат в архиве"
          members={members}
          resetKey={roomId ?? 0}
          roomId={roomId ?? undefined}
          showReceipts
          initialReads={initialReads}
          initialReactions={initialReactions}
          unreadCount={room?.unread ?? 0}
        />
      </div>
    </div>
  );
}
