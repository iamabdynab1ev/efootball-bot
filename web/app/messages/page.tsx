"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { ChevronLeft, MessageSquare } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { fetchRoomReactions, useChatRoom, useDirectRooms, type DirectRoomView, type ReactionAgg } from "@/lib/chat";
import { usePresence } from "@/lib/presence";
import { ChatThread } from "@/components/ChatThread";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

// «был(а) в сети …» для офлайна.
function lastSeenText(iso?: string): string {
  if (!iso) return "не в сети";
  const d = new Date(iso);
  const today = new Date();
  const yest = new Date(); yest.setDate(today.getDate() - 1);
  const hhmm = d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === today.toDateString()) return `был(а) в ${hhmm}`;
  if (d.toDateString() === yest.toDateString()) return `был(а) вчера в ${hhmm}`;
  return `был(а) ${d.toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" })}`;
}

function fmtWhen(iso?: string) {
  if (!iso) return "";
  const d = new Date(iso);
  const today = new Date();
  if (d.toDateString() === today.toDateString()) {
    return d.toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
  }
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "2-digit" });
}

// Открытый диалог — полноэкранный оверлей на мобиле (как чат турнира).
function Thread({ roomId, conv }: { roomId: number; conv: DirectRoomView | null }) {
  const { user } = useAuth();
  const router = useRouter();
  const { messages, loading: msgsLoading, hasMore, send, sendVoice, sendPhoto, loadOlder } = useChatRoom(roomId);
  const { isOnline } = usePresence();
  const [peerTyping, setPeerTyping] = useState(false);
  const [reactions, setReactions] = useState<ReactionAgg[]>([]);
  useEffect(() => {
    let on = true;
    fetchRoomReactions(roomId).then((r) => { if (on) setReactions(r); }).catch(() => { /* без реакций */ });
    return () => { on = false; };
  }, [roomId]);

  const otherName = conv?.other_name ?? "";
  const otherId = conv?.other_id;
  const initialReads = useMemo(
    () => (conv ? { [conv.other_id]: conv.other_last_read } : undefined),
    [conv?.other_id, conv?.other_last_read],
  );
  const goBack = () => router.push("/messages");

  const status = peerTyping
    ? "печатает…"
    : otherId && isOnline(otherId)
      ? "в сети"
      : lastSeenText(conv?.other_last_seen);

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-zinc-950 lg:static lg:z-auto lg:-my-8 lg:h-[calc(100dvh-2rem)] lg:min-h-[440px]">
      <header className="flex items-center gap-2.5 px-3 py-2.5 flex-shrink-0 border-b border-white/5 bg-zinc-950/95 backdrop-blur-sm shadow-sm shadow-black/20 pt-[max(0.625rem,env(safe-area-inset-top))] lg:border-0 lg:bg-transparent lg:px-0 lg:pt-0">
        <button
          onClick={goBack}
          aria-label="Назад"
          className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors -ml-1"
        >
          <ChevronLeft size={22} />
        </button>
        <div className="relative flex-shrink-0">
          <PlayerAvatar displayName={otherName || "?"} favoriteClub={conv?.other_club} size={34} />
          {otherId && isOnline(otherId) && (
            <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-zinc-950 bg-green-400" />
          )}
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-bold text-zinc-100 leading-tight">{otherName || "Диалог"}</p>
          <p className={cn("text-[11px] leading-tight truncate", peerTyping ? "text-yellow-400" : "text-zinc-500")}>{status}</p>
        </div>
      </header>

      <div className="flex-1 min-h-0">
        <ChatThread
          key={roomId}
          messages={messages}
          loading={msgsLoading}
          hasMore={hasMore}
          loadOlder={loadOlder}
          send={send}
          sendVoice={sendVoice}
          sendPhoto={sendPhoto}
          currentUserId={user?.id}
          isAdmin={user?.is_admin}
          showAuthorNames={false}
          resetKey={roomId}
          roomId={roomId}
          showReceipts
          initialReads={initialReads}
          initialReactions={reactions}
          unreadCount={conv?.unread ?? 0}
          showTyping
          onPeerTyping={setPeerTyping}
        />
      </div>
    </div>
  );
}

function MessagesInner() {
  const { user } = useAuth();
  const searchParams = useSearchParams();
  const { rooms, loading } = useDirectRooms();
  const { isOnline } = usePresence();
  const roomId = Number(searchParams.get("room")) || null;

  const active = useMemo(() => rooms.find((r) => r.room_id === roomId) ?? null, [rooms, roomId]);

  if (!user) {
    return (
      <div className="py-16 text-center">
        <MessageSquare size={26} className="mx-auto mb-3 text-zinc-600" />
        <p className="text-sm text-zinc-500">Войдите, чтобы читать личные сообщения.</p>
        <Link href="/login" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-900">Войти</Link>
      </div>
    );
  }

  // Открытый диалог (в т.ч. по deep-link из уведомления).
  if (roomId) {
    return <Thread roomId={roomId} conv={active} />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-yellow-400/15 text-yellow-400">
          <MessageSquare size={18} />
        </div>
        <div>
          <h1 className="text-lg font-bold text-zinc-100 leading-tight">Сообщения</h1>
          <p className="text-[11px] text-zinc-500">Личные чаты с соперниками</p>
        </div>
      </div>

      {loading ? (
        <div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>
      ) : rooms.length === 0 ? (
        <div className="rounded-xl border border-zinc-800 bg-zinc-900 px-4 py-12 text-center">
          <MessageSquare size={24} className="mx-auto mb-3 text-zinc-600" />
          <p className="text-sm text-zinc-400 font-medium">Пока нет диалогов</p>
          <p className="mt-1 text-xs text-zinc-500">Откройте матч и нажмите «Написать сопернику», чтобы начать личный чат.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900 divide-y divide-zinc-800/60">
          {rooms.map((r) => (
            <Link
              key={r.room_id}
              href={`/messages?room=${r.room_id}`}
              className="flex items-center gap-3 px-3.5 py-3 hover:bg-zinc-800/40 transition-colors"
            >
              <div className="relative flex-shrink-0">
                <PlayerAvatar displayName={r.other_name} favoriteClub={r.other_club} size={44} />
                {isOnline(r.other_id) && (
                  <span className="absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-zinc-900 bg-green-400" />
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className={cn("truncate text-sm font-semibold", r.unread > 0 ? "text-zinc-50" : "text-zinc-100")}>{r.other_name}</p>
                  <span className="flex-shrink-0 text-[10px] text-zinc-500">{fmtWhen(r.last_at)}</span>
                </div>
                <div className="flex items-center justify-between gap-2 mt-0.5">
                  <p className={cn("truncate text-xs", r.unread > 0 ? "text-zinc-300 font-medium" : "text-zinc-500")}>
                    {r.last_author_id === user.id ? "Вы: " : ""}{r.last_body || "Нет сообщений"}
                  </p>
                  {r.unread > 0 && (
                    <span className="flex h-5 min-w-[20px] flex-shrink-0 items-center justify-center rounded-full bg-yellow-400 px-1.5 text-[11px] font-bold text-zinc-900">
                      {r.unread > 99 ? "99+" : r.unread}
                    </span>
                  )}
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">Загрузка…</div>}>
      <MessagesInner />
    </Suspense>
  );
}
