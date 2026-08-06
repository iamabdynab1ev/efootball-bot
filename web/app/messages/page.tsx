"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { AnimatePresence, m } from "framer-motion";
import { ChevronLeft, MessageSquare, MoreVertical, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { useAuth } from "@/lib/auth";
import { DATE_LOCALES, tr, useLang, type Lang } from "@/lib/i18n";
import { deleteDirectChat, fetchRoomReactions, useChatRoom, useDirectRooms, type DirectRoomView, type ReactionAgg } from "@/lib/chat";
import { usePresence } from "@/lib/presence";
import { useSSE } from "@/lib/sse";
import { ChatThread } from "@/components/ChatThread";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

// «был(а) в сети …» для офлайна.
function lastSeenText(iso: string | undefined, lang: Lang, tr: { offline: string; wasAt: string; wasYesterdayAt: string; wasOn: string }): string {
  if (!iso) return tr.offline;
  const d = new Date(iso);
  const today = new Date();
  const yest = new Date(); yest.setDate(today.getDate() - 1);
  const hhmm = d.toLocaleTimeString(DATE_LOCALES[lang], { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === today.toDateString()) return `${tr.wasAt} ${hhmm}`;
  if (d.toDateString() === yest.toDateString()) return `${tr.wasYesterdayAt} ${hhmm}`;
  return `${tr.wasOn} ${d.toLocaleDateString(DATE_LOCALES[lang], { day: "2-digit", month: "2-digit" })}`;
}

// «печатает» + три прыгающие точки (анимация в globals.css).
function TypingDots({ label }: { label: string }) {
  return (
    <span className="inline-flex items-baseline gap-[1px]">
      {label}
      <span className="ml-0.5 inline-flex items-center gap-[2px]">
        {[0, 1, 2].map((i) => (
          <span key={i} className="typing-dot inline-block h-[3px] w-[3px] rounded-full bg-yellow-400" />
        ))}
      </span>
    </span>
  );
}

function fmtWhen(iso: string | undefined, lang: Lang) {
  if (!iso) return "";
  const d = new Date(iso);
  const today = new Date();
  if (d.toDateString() === today.toDateString()) {
    return d.toLocaleTimeString(DATE_LOCALES[lang], { hour: "2-digit", minute: "2-digit" });
  }
  return d.toLocaleDateString(DATE_LOCALES[lang], { day: "2-digit", month: "2-digit" });
}

// Шторка удаления чата — как в мессенджерах: «у меня» / «у обоих» (двойное
// подтверждение для необратимого варианта).
function DeleteSheet({ conv, onClose, onDeleted }: {
  conv: DirectRoomView;
  onClose: () => void;
  onDeleted?: () => void;
}) {
  const { t } = useLang();
  const [confirmBoth, setConfirmBoth] = useState(false);
  const [busy, setBusy] = useState(false);

  const del = async (forBoth: boolean) => {
    if (forBoth && !confirmBoth) { setConfirmBoth(true); return; }
    setBusy(true);
    try {
      await deleteDirectChat(conv.room_id, forBoth);
      toast.success(forBoth ? t("messages.deletedBoth") : t("messages.deletedMine"));
      onClose();
      onDeleted?.();
    } catch {
      toast.error(t("messages.deleteFail"));
      setBusy(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[60]" role="dialog" aria-modal="true" aria-label={t("messages.deleteTitle")}>
      <m.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.18 }}
        className="absolute inset-0 bg-black/60"
        onClick={onClose}
      />
      {/* Позиционирование — на обёртке, анимация — на внутреннем слое:
          иначе framer перезаписал бы -translate-x/y центровки на десктопе. */}
      <div className="absolute inset-x-0 bottom-0 lg:inset-x-auto lg:bottom-auto lg:left-1/2 lg:top-1/2 lg:w-[380px] lg:-translate-x-1/2 lg:-translate-y-1/2">
      <m.div
        initial={{ y: "100%", opacity: 0.6 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: "100%", opacity: 0.6 }}
        transition={{ type: "spring", stiffness: 380, damping: 36 }}
        className="rounded-t-2xl border-t border-zinc-800 bg-zinc-900 p-4 pb-[max(1rem,env(safe-area-inset-bottom))] lg:rounded-2xl lg:border lg:pb-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-red-500/15 text-red-400">
            <Trash2 size={18} />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-bold text-zinc-100">{t("messages.deleteWith")} {conv.other_name}?</p>
            <p className="mt-0.5 text-[11px] text-zinc-500">
              {t("messages.deleteMineDesc")} {t("messages.deleteBothDesc")}
            </p>
          </div>
        </div>
        <div className="mt-4 space-y-2">
          <button
            onClick={() => del(false)}
            disabled={busy}
            className="w-full rounded-lg border border-zinc-700 py-2.5 text-sm font-semibold text-zinc-200 hover:bg-zinc-800 transition-colors disabled:opacity-50"
          >
            {t("messages.deleteMine")}
          </button>
          <button
            onClick={() => del(true)}
            disabled={busy}
            className={cn(
              "w-full rounded-lg py-2.5 text-sm font-bold transition-colors disabled:opacity-50",
              confirmBoth ? "bg-red-500 text-white" : "border border-red-500/40 text-red-400 hover:bg-red-500/10",
            )}
          >
            {confirmBoth ? t("messages.confirmBoth") : t("messages.deleteBoth")}
          </button>
          <button
            onClick={onClose}
            className="w-full rounded-lg py-2.5 text-sm font-semibold text-zinc-400 hover:text-zinc-200 transition-colors"
          >
            {t("messages.cancel")}
          </button>
        </div>
      </m.div>
      </div>
    </div>
  );
}

// Открытый диалог — полноэкранный оверлей на мобиле (как чат турнира).
function Thread({ roomId, conv }: { roomId: number; conv: DirectRoomView | null }) {
  const { user } = useAuth();
  const router = useRouter();
  const { messages, loading: msgsLoading, hasMore, send, sendVoice, sendPhoto, loadOlder } = useChatRoom(roomId);
  const { isOnline } = usePresence();
  const { t, lang } = useLang();
  const [peerTyping, setPeerTyping] = useState(false);
  const [showDelete, setShowDelete] = useState(false);
  const [reactions, setReactions] = useState<ReactionAgg[]>([]);

  // Собеседник удалил чат «у обоих», пока диалог открыт — закрываем его.
  useSSE("chat_cleared", (d: any) => {
    if (d?.room_id === roomId && d?.for_both && d?.by !== user?.id) {
      toast(t("messages.otherDeleted"));
      router.replace("/messages");
    }
  }, true);
  useEffect(() => {
    let on = true;
    fetchRoomReactions(roomId).then((r) => { if (on) setReactions(r); }).catch(() => { /* без реакций */ });
    return () => { on = false; };
  }, [roomId]);

  const otherName = conv?.other_name ?? "";
  const otherId = conv?.other_id;
  const otherLastRead = conv?.other_last_read;
  const initialReads = useMemo(
    () => (otherId != null && otherLastRead != null ? { [otherId]: otherLastRead } : undefined),
    [otherId, otherLastRead],
  );
  const goBack = () => router.push("/messages");

  const status = peerTyping
    ? <TypingDots label={t("messages.typing")} />
    : otherId && isOnline(otherId)
      ? t("messages.online")
      : lastSeenText(conv?.other_last_seen, lang, { offline: t("messages.offline"), wasAt: t("messages.wasAt"), wasYesterdayAt: t("messages.wasYesterdayAt"), wasOn: t("messages.wasOn") });

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-zinc-950 lg:static lg:z-auto lg:-my-8 lg:h-[calc(100dvh-2rem)] lg:min-h-[440px]">
      <header className="flex items-center gap-2.5 px-3 py-2.5 flex-shrink-0 border-b border-white/5 bg-zinc-950/95 backdrop-blur-sm shadow-sm shadow-black/20 pt-[max(0.625rem,env(safe-area-inset-top))] lg:border-0 lg:bg-transparent lg:px-0 lg:pt-0">
        <button
          onClick={goBack}
          aria-label={t("messages.back")}
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
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-bold text-zinc-100 leading-tight">{otherName || t("messages.dialog")}</p>
          <p className={cn("text-[11px] leading-tight truncate", peerTyping ? "text-yellow-400" : "text-zinc-500")}>{status}</p>
        </div>
        {conv && (
          <button
            onClick={() => setShowDelete(true)}
            aria-label={t("messages.chatActions")}
            className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100 transition-colors"
          >
            <MoreVertical size={18} />
          </button>
        )}
      </header>
      <AnimatePresence>
        {showDelete && conv && (
          <DeleteSheet conv={conv} onClose={() => setShowDelete(false)} onDeleted={goBack} />
        )}
      </AnimatePresence>

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
  const { t, lang } = useLang();
  const { isOnline } = usePresence();
  const roomId = Number(searchParams.get("room")) || null;
  const [sheetFor, setSheetFor] = useState<DirectRoomView | null>(null);

  const active = useMemo(() => rooms.find((r) => r.room_id === roomId) ?? null, [rooms, roomId]);

  if (!user) {
    return (
      <div className="py-16 text-center">
        <MessageSquare size={26} className="mx-auto mb-3 text-zinc-600" />
        <p className="text-sm text-zinc-500">{t("messages.loginPrompt")}</p>
        <Link href="/login" className="mt-4 inline-block rounded-lg bg-yellow-400 px-4 py-2 text-sm font-bold text-zinc-900">{t("nav.login")}</Link>
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
          <h1 className="text-lg font-bold text-zinc-100 leading-tight">{t("messages.title")}</h1>
          <p className="text-[11px] text-zinc-500">{t("messages.subtitle")}</p>
        </div>
      </div>

      {loading ? (
        /* Skeleton списка диалогов — без текстовых заглушек и прыжков высоты */
        <div className="divide-y divide-zinc-800/60 overflow-hidden rounded-xl card-premium" aria-hidden>
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="flex items-center gap-3 px-3.5 py-3">
              <span className="skeleton h-11 w-11 flex-shrink-0 rounded-full" />
              <div className="min-w-0 flex-1 space-y-2">
                <span className="skeleton block h-3.5 w-32 rounded" />
                <span className="skeleton block h-3 w-48 max-w-full rounded" />
              </div>
            </div>
          ))}
        </div>
      ) : rooms.length === 0 ? (
        <div className="rounded-xl card-premium px-4 py-12 text-center">
          <MessageSquare size={24} className="mx-auto mb-3 text-zinc-600" />
          <p className="text-sm text-zinc-400 font-medium">{t("messages.empty")}</p>
          <p className="mt-1 text-xs text-zinc-500">{t("messages.emptyDesc")}</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl card-premium divide-y divide-zinc-800/60">
          {rooms.map((r) => (
            <Link
              key={r.room_id}
              href={`/messages?room=${r.room_id}`}
              className="pressable flex items-center gap-3 px-3.5 py-3 hover:bg-zinc-800/40 active:bg-zinc-800/60"
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
                  <span className="flex-shrink-0 text-[10px] text-zinc-500">{fmtWhen(r.last_at, lang)}</span>
                </div>
                <div className="flex items-center justify-between gap-2 mt-0.5">
                  <p className={cn("truncate text-xs", r.unread > 0 ? "text-zinc-300 font-medium" : "text-zinc-500")}>
                    {r.last_author_id === user.id ? t("messages.youPrefix") : ""}{r.last_body || (r.last_at ? t("messages.attachment") : t("messages.noMessages"))}
                  </p>
                  {r.unread > 0 && (
                    <span className="flex h-5 min-w-[20px] flex-shrink-0 items-center justify-center rounded-full bg-yellow-400 px-1.5 text-[11px] font-bold text-zinc-900">
                      {r.unread > 99 ? "99+" : r.unread}
                    </span>
                  )}
                </div>
              </div>
              <button
                onClick={(e) => { e.preventDefault(); e.stopPropagation(); setSheetFor(r); }}
                aria-label={`${t("messages.chatActions")}: ${r.other_name}`}
                className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg text-zinc-500 hover:bg-zinc-800 hover:text-zinc-200 transition-colors"
              >
                <MoreVertical size={16} />
              </button>
            </Link>
          ))}
        </div>
      )}

      <AnimatePresence>
        {sheetFor && <DeleteSheet conv={sheetFor} onClose={() => setSheetFor(null)} />}
      </AnimatePresence>
    </div>
  );
}

export default function Page() {
  return (
    <Suspense fallback={<div className="py-10 text-center text-sm text-zinc-500">{tr("common.loading")}</div>}>
      <MessagesInner />
    </Suspense>
  );
}
