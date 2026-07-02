"use client";

import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Send, Trash2, ChevronDown, Check, CheckCheck, Pencil, X, Reply, SmilePlus, Mic, Play, Pause, ImagePlus, Copy } from "lucide-react";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { markRead, sendTyping, deleteChatMessage, editChatMessage, addReaction, removeReaction, type ChatMessage, type ChatMember, type ReactionAgg } from "@/lib/chat";
import { useSSE } from "@/lib/sse";
import { PlayerAvatar } from "@/components/PlayerAvatar";
import { cn } from "@/lib/utils";

function fmtTime(iso: string) {
  return new Date(iso).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" });
}

function dayLabel(iso: string) {
  const d = new Date(iso);
  const today = new Date();
  const yest = new Date(); yest.setDate(today.getDate() - 1);
  if (d.toDateString() === today.toDateString()) return "Сегодня";
  if (d.toDateString() === yest.toDateString()) return "Вчера";
  return d.toLocaleDateString("ru-RU", { day: "2-digit", month: "long" });
}

// Подсветка @упоминаний.
function renderBody(body: string) {
  return body.split(/(@[\wА-Яа-яЁё]+)/g).map((p, i) =>
    p.startsWith("@")
      ? <span key={i} className="font-semibold text-yellow-400">{p}</span>
      : <span key={i}>{p}</span>
  );
}

const REACTION_EMOJIS = ["👍", "❤️", "😂", "🔥", "👏", "😮", "😢"];
const EMPTY_REACTIONS: ReactionAgg[] = []; // стабильная ссылка — не ломает memo

function fmtDur(sec: number) {
  const s = Math.max(0, Math.floor(sec || 0));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

// Псевдо-waveform: детерминированные высоты столбиков по url (без декодирования).
function waveHeights(seed: string, n = 30): number[] {
  let h = 2166136261 >>> 0;
  for (let i = 0; i < seed.length; i++) { h ^= seed.charCodeAt(i); h = Math.imul(h, 16777619) >>> 0; }
  const out: number[] = [];
  for (let i = 0; i < n; i++) { h = (Math.imul(h, 1103515245) + 12345) >>> 0; out.push(0.28 + (h % 1000) / 1000 * 0.72); }
  return out;
}

// Плеер голосового: waveform (кликабельный seek) + скорость 1×/1.5×/2×.
const VoiceMessage = memo(function VoiceMessage({ url, dur, own }: { url: string; dur?: number; own: boolean }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);
  const [cur, setCur] = useState(0);
  const [total, setTotal] = useState(dur || 0);
  const [rate, setRate] = useState(1);
  const bars = useMemo(() => waveHeights(url), [url]);
  const progress = total > 0 ? cur / total : 0;

  const toggle = () => {
    const a = audioRef.current;
    if (!a) return;
    if (playing) a.pause(); else a.play().catch(() => { /* автоплей заблокирован */ });
  };
  const cycleRate = () => {
    const next = rate === 1 ? 1.5 : rate === 1.5 ? 2 : 1;
    setRate(next);
    if (audioRef.current) audioRef.current.playbackRate = next;
  };
  const seek = (e: React.MouseEvent<HTMLDivElement>) => {
    const a = audioRef.current;
    if (!a || !total) return;
    const rect = e.currentTarget.getBoundingClientRect();
    const frac = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
    a.currentTime = frac * total;
    setCur(a.currentTime);
  };

  return (
    <div className="flex items-center gap-2.5 min-w-[210px] py-0.5">
      <button
        onClick={toggle}
        aria-label={playing ? "Пауза" : "Играть"}
        className={cn("flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full", own ? "bg-yellow-400 text-zinc-900" : "bg-white/10 text-zinc-100")}
      >
        {playing ? <Pause size={16} /> : <Play size={16} className="ml-0.5" />}
      </button>
      <div className="min-w-0 flex-1">
        <div onClick={seek} className="flex h-7 cursor-pointer items-center gap-[2px]">
          {bars.map((hgt, i) => {
            const filled = i / bars.length <= progress;
            return (
              <span
                key={i}
                className={cn("w-[3px] flex-1 rounded-full transition-colors", filled ? (own ? "bg-yellow-400" : "bg-sky-400") : "bg-white/20")}
                style={{ height: `${Math.round(hgt * 100)}%` }}
              />
            );
          })}
        </div>
        <div className="mt-0.5 text-[10px] tabular-nums text-zinc-400">{fmtDur(playing || cur ? cur : total)}</div>
      </div>
      <button
        onClick={cycleRate}
        aria-label="Скорость воспроизведения"
        className="flex-shrink-0 rounded-md bg-white/10 px-1.5 py-0.5 text-[10px] font-bold text-zinc-200 tabular-nums hover:bg-white/20"
      >
        {rate}×
      </button>
      <audio
        ref={audioRef}
        src={url}
        preload="metadata"
        onPlay={() => setPlaying(true)}
        onPause={() => setPlaying(false)}
        onEnded={() => { setPlaying(false); setCur(0); }}
        onTimeUpdate={(e) => setCur(e.currentTarget.currentTime)}
        onLoadedMetadata={(e) => { const d = e.currentTarget.duration; if (isFinite(d) && d > 0) setTotal(d); }}
      />
    </div>
  );
});

// Группируем плоский список реакций в карту {msgId: агрегаты}.
function groupReactions(list?: ReactionAgg[]): Record<number, ReactionAgg[]> {
  const map: Record<number, ReactionAgg[]> = {};
  for (const r of list ?? []) (map[r.message_id] ??= []).push(r);
  return map;
}

// Лайтбокс: просмотр фото на весь экран, тап/Esc — закрыть.
function Lightbox({ url, onClose }: { url: string; onClose: () => void }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);
  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/92 p-4" onClick={onClose}>
      <button onClick={onClose} aria-label="Закрыть" className="absolute right-4 top-[max(1rem,env(safe-area-inset-top))] rounded-full bg-white/10 p-2 text-white hover:bg-white/20">
        <X size={20} />
      </button>
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img src={url} alt="фото" className="max-h-full max-w-full rounded-lg object-contain" onClick={(e) => e.stopPropagation()} />
    </div>
  );
}

interface RowData {
  m: ChatMessage;
  own: boolean;
  grouped: boolean;  // подряд от того же автора в пределах 5 мин — тесная группа
  showDay: boolean;  // сменился день — показать разделитель
  day: string;
  showName: boolean; // показывать имя автора (в ЛС/у своих не нужно)
  replyAuthor?: string; // превью сообщения, на которое отвечают
  replyBody?: string;
}

type RowExtra = {
  isAdmin: boolean;
  onDelete: (id: number, own: boolean) => void;
  onEdit: (m: ChatMessage) => void;
  onReply: (m: ChatMessage) => void;
  onReact: (id: number) => void;
  onPickEmoji: (id: number, emoji: string) => void;
  onToggleReaction: (id: number, emoji: string, mine: boolean) => void;
  onImageClick: (url: string) => void;
  onReplyClick: (id: number) => void;
  onLongPress: (m: ChatMessage) => void;
  onQuickLike: (m: ChatMessage) => void;
  showReceipts: boolean;
  otherReads: number[];
  reactions: ReactionAgg[];
  pickerOpen: boolean;
  showUnread: boolean; // разделитель «непрочитанные» перед этим сообщением
};

const isInteractive = (t: EventTarget | null) => !!(t as HTMLElement)?.closest?.("button, a, audio, input, textarea, img");

// MessageRow мемоизирован: при новом сообщении перерисовывается только оно, а не
// весь список (объекты старых сообщений сохраняют идентичность).
const MessageRow = memo(function MessageRow({
  m, own, grouped, showDay, day, showName, replyAuthor, replyBody,
  isAdmin, onDelete, onEdit, onReply, onReact, onPickEmoji, onToggleReaction,
  onImageClick, onReplyClick, onLongPress, onQuickLike,
  showReceipts, otherReads, reactions, pickerOpen, showUnread,
}: RowData & RowExtra) {
  const receipts = own && showReceipts && !m.deleted;
  const total = otherReads.length;
  const readers = receipts ? otherReads.filter((v) => v >= m.id).length : 0;
  const allRead = receipts && total > 0 && readers === total;
  const imageOnly = !m.deleted && m.media?.type === "image" && !m.body && !replyAuthor && !(showName && !grouped);

  // Жесты: свайп-вправо → ответить, долгое нажатие → меню, двойной тап → ❤️.
  const [dx, setDx] = useState(0);
  const g = useRef({ x: 0, y: 0, swiping: false, lp: 0 as any, lastTap: 0 });
  const clearLP = () => { if (g.current.lp) { clearTimeout(g.current.lp); g.current.lp = 0; } };
  const onTouchStart = (e: React.TouchEvent) => {
    const t = e.touches[0];
    g.current.x = t.clientX; g.current.y = t.clientY; g.current.swiping = false;
    if (!m.deleted && !isInteractive(e.target)) {
      g.current.lp = setTimeout(() => { g.current.lp = 0; onLongPress(m); }, 480);
    }
  };
  const onTouchMove = (e: React.TouchEvent) => {
    const t = e.touches[0];
    const ddx = t.clientX - g.current.x, ddy = t.clientY - g.current.y;
    if (!g.current.swiping && Math.abs(ddx) > 12 && Math.abs(ddx) > Math.abs(ddy) + 4) { g.current.swiping = true; clearLP(); }
    if (g.current.swiping) setDx(Math.max(0, Math.min(64, ddx)));
  };
  const onTouchEnd = (e: React.TouchEvent) => {
    clearLP();
    if (g.current.swiping) { if (dx > 44) onReply(m); setDx(0); return; }
    if (!m.deleted && !isInteractive(e.target)) {
      const now = Date.now();
      if (now - g.current.lastTap < 300) { onQuickLike(m); g.current.lastTap = 0; }
      else g.current.lastTap = now;
    }
  };

  return (
    <div id={`msg-${m.id}`} className="scroll-mt-16">
      {showDay && (
        <div className="flex justify-center py-2.5">
          <span className="chat-pill rounded-full px-3 py-1 text-[10px] font-semibold tracking-wide text-zinc-400">{day}</span>
        </div>
      )}
      {showUnread && (
        <div className="flex items-center gap-2 py-2">
          <span className="h-px flex-1 bg-yellow-400/25" />
          <span className="text-[10px] font-semibold uppercase tracking-wider text-yellow-400/80">Непрочитанные</span>
          <span className="h-px flex-1 bg-yellow-400/25" />
        </div>
      )}
      <div
        className={cn("relative msg-in flex items-end gap-2 group", own ? "flex-row-reverse" : "", grouped ? "mt-0.5" : "mt-2.5")}
        onTouchStart={onTouchStart} onTouchMove={onTouchMove} onTouchEnd={onTouchEnd}
      >
        {/* Подсказка ответа при свайпе */}
        {dx > 4 && (
          <span className="pointer-events-none absolute left-1 top-1/2 -translate-y-1/2 text-yellow-400" style={{ opacity: Math.min(1, dx / 44) }}>
            <Reply size={18} />
          </span>
        )}
        {grouped
          ? <div className="w-[26px] flex-shrink-0" />
          : <PlayerAvatar displayName={m.author_name} favoriteClub={m.author_club} size={26} />}

        <div
          className={cn("flex min-w-0 max-w-[85%] flex-col sm:max-w-[70%]", own ? "items-end" : "items-start", reactions.length > 0 && "mb-2.5")}
          style={dx ? { transform: `translateX(${dx}px)` } : { transition: "transform 0.18s ease" }}
        >
          <div className={cn(
            "relative w-fit max-w-full rounded-2xl shadow-sm shadow-black/20",
            imageOnly ? "overflow-hidden p-0" : "px-3.5 py-2",
            own ? "bubble-out rounded-br-md" : "bubble-in rounded-bl-md",
            grouped && (own ? "rounded-tr-md" : "rounded-tl-md"),
          )}>
            {showName && !grouped && (
              <p className={cn("text-[11px] font-semibold mb-0.5", own ? "text-right text-yellow-300/90" : "text-yellow-400/90")}>
                {own ? "Вы" : m.author_name}
              </p>
            )}
            {replyAuthor && !m.deleted && (
              <button
                onClick={() => m.reply_to_id && onReplyClick(m.reply_to_id)}
                className="mb-1 block w-full max-w-full rounded-md border-l-2 border-yellow-400/60 bg-black/20 px-2 py-1 text-left hover:bg-black/30"
              >
                <p className="truncate text-[11px] font-semibold text-yellow-300/90">{replyAuthor}</p>
                <p className="truncate text-[11px] text-zinc-400">{replyBody}</p>
              </button>
            )}
            {m.deleted ? (
              <p className="text-xs italic text-zinc-500">сообщение удалено</p>
            ) : (
              <>
                {m.media?.type === "audio" && <VoiceMessage url={m.media.url} dur={m.media.dur} own={own} />}
                {m.media?.type === "image" && (
                  <button onClick={() => onImageClick(m.media!.url)} className={cn("block overflow-hidden", imageOnly ? "" : "mb-1 rounded-lg")}>
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img src={m.media.url} alt="фото" loading="lazy" className={cn("object-cover", imageOnly ? "max-h-80 w-full" : "max-h-72 w-auto max-w-full")} />
                  </button>
                )}
                {m.body && (
                  <p className="text-[15px] leading-snug text-zinc-100 whitespace-pre-wrap break-words [overflow-wrap:anywhere]">
                    {renderBody(m.body)}
                  </p>
                )}
              </>
            )}
            {/* Время + галочки: у фото-без-текста — оверлеем на снимке */}
            {imageOnly ? (
              <div className="absolute bottom-1.5 right-1.5 flex items-center gap-1 rounded-full bg-black/55 px-1.5 py-0.5 backdrop-blur-sm">
                <span className="text-[10px] text-zinc-100">{fmtTime(m.created_at)}</span>
                {receipts && (readers === 0 ? <Check size={12} className="text-zinc-300" /> : <CheckCheck size={12} className={allRead ? "text-sky-300" : "text-zinc-300"} />)}
              </div>
            ) : (
              <div className={cn("flex items-center gap-1 mt-0.5", own ? "justify-end" : "")}>
                {m.edited && !m.deleted && <span className="text-[10px] text-zinc-600 italic">изменено</span>}
                <span className="text-[10px] text-zinc-500">{fmtTime(m.created_at)}</span>
                {receipts && (
                  readers === 0
                    ? <Check size={13} className="text-zinc-500" />
                    : (
                      <span className={cn("flex items-center gap-0.5", allRead ? "text-sky-400" : "text-sky-400/60")}>
                        <CheckCheck size={13} />
                        {total > 1 && <span className="text-[9px] font-semibold">{readers}</span>}
                      </span>
                    )
                )}
              </div>
            )}

            {/* Реакции — на линии рамки пузыря (без своей рамки) */}
            {reactions.length > 0 && (
              <div className={cn("absolute -bottom-2.5 z-10 flex flex-wrap gap-1", own ? "right-2" : "left-2")}>
                {reactions.map((r) => (
                  <button
                    key={r.emoji}
                    onClick={() => onToggleReaction(m.id, r.emoji, r.mine)}
                    className={cn(
                      "flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-xs shadow-md shadow-black/40 transition-colors",
                      r.mine ? "bg-yellow-400 text-zinc-900" : "bg-zinc-800 text-zinc-100 hover:bg-zinc-700",
                    )}
                  >
                    <span>{r.emoji}</span>
                    <span className="text-[10px] font-bold">{r.count}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Выбор эмодзи (кнопка «реакция») */}
          {pickerOpen && !m.deleted && (
            <div className={cn("chat-pill mt-2 flex gap-1 rounded-full px-2 py-1", own ? "self-end" : "self-start")}>
              {REACTION_EMOJIS.map((e) => (
                <button key={e} onClick={() => onPickEmoji(m.id, e)} className="text-lg leading-none transition-transform hover:scale-125" aria-label={`Реакция ${e}`}>
                  {e}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Desktop-иконки действий (на мобиле — долгое нажатие) */}
        {!m.deleted && (
          <div className="hidden items-center gap-0.5 self-center opacity-0 transition-opacity group-hover:opacity-100 sm:flex">
            <button onClick={() => onReply(m)} className="rounded p-1 text-zinc-600 hover:text-yellow-400" title="Ответить"><Reply size={13} /></button>
            <button onClick={() => onReact(m.id)} className="rounded p-1 text-zinc-600 hover:text-yellow-400" title="Реакция"><SmilePlus size={13} /></button>
            {own && <button onClick={() => onEdit(m)} className="rounded p-1 text-zinc-600 hover:text-yellow-400" title="Редактировать"><Pencil size={13} /></button>}
            {(own || isAdmin) && <button onClick={() => onDelete(m.id, own)} className="rounded p-1 text-zinc-600 hover:text-red-400" title="Удалить"><Trash2 size={13} /></button>}
          </div>
        )}
      </div>
    </div>
  );
});

export interface ChatThreadProps {
  messages: ChatMessage[];
  hasMore: boolean;
  loadOlder: () => Promise<void> | void;
  send: (body: string, replyToId?: number) => Promise<void>;
  /** Отправка голосового (если поддерживается R2). */
  sendVoice?: (blob: Blob, dur: number) => Promise<void>;
  /** Отправка фото (если поддерживается R2). */
  sendPhoto?: (file: File) => Promise<void>;
  currentUserId?: number;
  isAdmin?: boolean;
  archived?: boolean;
  archivedNote?: string;
  /** Участники для @упоминаний (в ЛС не передаём — упоминания не нужны). */
  members?: ChatMember[];
  /** Показывать имя автора над чужими сообщениями (в ЛС не нужно). */
  showAuthorNames?: boolean;
  /** Плейсхолдер поля ввода. */
  placeholder?: string;
  /** Активна ли комната (для сброса скролла/ввода при переключении). */
  resetKey?: string | number;
  /** id комнаты — включает отметки прочтения и «печатает…» (для ЛС). */
  roomId?: number;
  /** Показывать ✓/✓✓ на своих сообщениях. */
  showReceipts?: boolean;
  /** Начальный прогресс прочтения участников {userId: lastRead} (для ✓✓). */
  initialReads?: Record<number, number>;
  /** Начальные реакции комнаты (агрегаты) — для отображения до первого события. */
  initialReactions?: ReactionAgg[];
  /** Непрочитанных на момент открытия (для разделителя «непрочитанные»). */
  unreadCount?: number;
  /** Слать/принимать «печатает…». */
  showTyping?: boolean;
  /** Колбэк: собеседник печатает / перестал (для статуса в шапке). */
  onPeerTyping?: (typing: boolean) => void;
}

// ChatThread — переиспользуемая лента диалога: сообщения (bottom-anchored),
// авто-скролл, кнопка «вниз», @упоминания, поле ввода. Используется и в чате
// турнира (на комнату), и в личных сообщениях.
export function ChatThread({
  messages, hasMore, loadOlder, send, sendVoice, sendPhoto,
  currentUserId, isAdmin = false, archived = false,
  archivedNote = "Чат в архиве", members = [], showAuthorNames = true,
  placeholder = "Сообщение…", resetKey,
  roomId, showReceipts = false, initialReads, initialReactions, unreadCount = 0, showTyping = false, onPeerTyping,
}: ChatThreadProps) {
  const [text, setText] = useState("");
  const [sending, setSending] = useState(false);
  const [mentionQuery, setMentionQuery] = useState<string | null>(null);
  const [showJump, setShowJump] = useState(false);
  const [editing, setEditing] = useState<{ id: number } | null>(null); // правка своего сообщения
  const [replyTo, setReplyTo] = useState<{ id: number; author: string; body: string } | null>(null); // ответ на сообщение
  const [reactPickerFor, setReactPickerFor] = useState<number | null>(null); // открыт выбор эмодзи для msg id
  const [reactionsByMsg, setReactionsByMsg] = useState<Record<number, ReactionAgg[]>>(() => groupReactions(initialReactions));
  const [recording, setRecording] = useState(false);   // идёт запись голосового
  const [recElapsed, setRecElapsed] = useState(0);      // секунды записи
  const [lightboxUrl, setLightboxUrl] = useState<string | null>(null);
  const [menuFor, setMenuFor] = useState<ChatMessage | null>(null); // долгое нажатие
  const [newCount, setNewCount] = useState(0);          // новых сообщений пока листаешь вверх
  const [firstUnreadId, setFirstUnreadId] = useState<number | null>(null);
  const unreadInitRef = useRef(false);
  const mediaRecRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const streamRef = useRef<MediaStream | null>(null);
  const recTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const recStartRef = useRef(0);
  // Прогресс прочтения по участникам {userId: lastRead}. Для ЛС — один собеседник,
  // для группы — все участники (сколько прочитали каждое сообщение).
  const [readsByUser, setReadsByUser] = useState<Record<number, number>>(initialReads ?? {});
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const typingSentRef = useRef(0);   // троттлинг отправки «печатает…»
  const readSentRef = useRef(0);     // до какого id уже отметили прочтение
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Начальный прогресс может приехать позже монтирования — вливаем по максимуму.
  useEffect(() => {
    if (!initialReads) return;
    setReadsByUser((prev) => {
      let changed = false;
      const next = { ...prev };
      for (const [uid, lr] of Object.entries(initialReads)) {
        if ((next[+uid] ?? 0) < lr) { next[+uid] = lr; changed = true; }
      }
      return changed ? next : prev;
    });
  }, [initialReads]);

  // Живое обновление ✓✓: кто-то из участников дочитал.
  useSSE("chat_read", (d: any) => {
    if (!showReceipts || roomId == null || !d || d.room_id !== roomId) return;
    const uid = Number(d.user_id) || 0;
    const lr = Number(d.last_read_id) || 0;
    if (!uid) return;
    setReadsByUser((prev) => (prev[uid] >= lr ? prev : { ...prev, [uid]: lr }));
  }, showReceipts && roomId != null);

  // Начальные реакции могут приехать позже монтирования — пересобираем карту.
  useEffect(() => {
    if (initialReactions) setReactionsByMsg(groupReactions(initialReactions));
  }, [initialReactions]);

  // Живое обновление реакций.
  useSSE("chat_reaction", (d: any) => {
    if (roomId == null || !d || d.room_id !== roomId) return;
    const mid = Number(d.message_id) || 0;
    const emoji = String(d.emoji || "");
    const mine = Number(d.user_id) === currentUserId;
    if (!mid || !emoji) return;
    setReactionsByMsg((prev) => {
      const arr = [...(prev[mid] ?? [])];
      const i = arr.findIndex((r) => r.emoji === emoji);
      if (d.op === "add") {
        if (i >= 0) arr[i] = { ...arr[i], count: arr[i].count + 1, mine: arr[i].mine || mine };
        else arr.push({ message_id: mid, emoji, count: 1, mine });
      } else {
        if (i >= 0) {
          const c = arr[i].count - 1;
          if (c <= 0) arr.splice(i, 1);
          else arr[i] = { ...arr[i], count: c, mine: mine ? false : arr[i].mine };
        }
      }
      return { ...prev, [mid]: arr };
    });
  }, roomId != null);

  const onReply = useCallback((m: ChatMessage) => {
    setReplyTo({ id: m.id, author: m.user_id === currentUserId ? "Вы" : m.author_name, body: m.deleted ? "сообщение удалено" : m.body });
    setReactPickerFor(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  }, [currentUserId]);

  const onReact = useCallback((id: number) => {
    setReactPickerFor((cur) => (cur === id ? null : id));
  }, []);

  const toggleReaction = useCallback((id: number, emoji: string, mine: boolean) => {
    setReactPickerFor(null);
    (mine ? removeReaction(id, emoji) : addReaction(id, emoji)).catch(() => { /* повторит по SSE/следующему тапу */ });
  }, []);

  const onPickEmoji = useCallback((id: number, emoji: string) => {
    const mine = (reactionsByMsg[id] ?? []).some((r) => r.emoji === emoji && r.mine);
    toggleReaction(id, emoji, mine);
  }, [reactionsByMsg, toggleReaction]);

  const onImageClick = useCallback((url: string) => setLightboxUrl(url), []);
  const onLongPress = useCallback((m: ChatMessage) => setMenuFor(m), []);
  const onQuickLike = useCallback((m: ChatMessage) => {
    const mine = (reactionsByMsg[m.id] ?? []).some((r) => r.emoji === "❤️" && r.mine);
    toggleReaction(m.id, "❤️", mine);
  }, [reactionsByMsg, toggleReaction]);
  const onReplyClick = useCallback((id: number) => {
    const el = document.getElementById(`msg-${id}`);
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    el.classList.add("row-flash");
    setTimeout(() => el.classList.remove("row-flash"), 2400);
  }, []);

  // Разделитель «непрочитанные»: фиксируем один раз при открытии.
  useEffect(() => {
    if (unreadInitRef.current) return;
    if (messages.length === 0 || unreadCount <= 0) return;
    unreadInitRef.current = true;
    const idx = Math.max(0, messages.length - unreadCount);
    setFirstUnreadId(messages[idx]?.id ?? null);
  }, [messages, unreadCount]);

  // Массив last_read остальных участников (без себя) — для подсчёта прочтений.
  const otherReads = useMemo(
    () => Object.entries(readsByUser)
      .filter(([uid]) => Number(uid) !== currentUserId)
      .map(([, lr]) => lr),
    [readsByUser, currentUserId],
  );

  // «печатает…»: показываем и гасим через 3.5с тишины.
  useSSE("chat_typing", (d: any) => {
    if (!showTyping || roomId == null || !d || d.room_id !== roomId) return;
    onPeerTyping?.(true);
    if (typingTimerRef.current) clearTimeout(typingTimerRef.current);
    typingTimerRef.current = setTimeout(() => onPeerTyping?.(false), 3500);
  }, showTyping && roomId != null);

  useEffect(() => () => { if (typingTimerRef.current) clearTimeout(typingTimerRef.current); }, []);

  // Отмечаем прочитанным до последнего сообщения (при открытии и на новые входящие).
  useEffect(() => {
    if (roomId == null || messages.length === 0) return;
    const lastId = messages[messages.length - 1].id;
    if (lastId > readSentRef.current) {
      readSentRef.current = lastId;
      markRead(roomId, lastId);
    }
  }, [roomId, messages]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const lastIdRef = useRef(0);
  const stickRef = useRef(true);
  const prevHeightRef = useRef(0);

  const rows = useMemo<RowData[]>(() => {
    const byId = new Map(messages.map((x) => [x.id, x]));
    const out: RowData[] = [];
    let prevAuthor: number | null | undefined;
    let prevDay = "";
    let prevTime = 0;
    for (const m of messages) {
      const day = dayLabel(m.created_at);
      const t = new Date(m.created_at).getTime();
      const grouped = m.user_id === prevAuthor && day === prevDay && t - prevTime < 5 * 60 * 1000;
      let replyAuthor: string | undefined;
      let replyBody: string | undefined;
      if (m.reply_to_id) {
        const rt = byId.get(m.reply_to_id);
        replyAuthor = rt ? (rt.user_id === currentUserId ? "Вы" : rt.author_name) : "Сообщение";
        replyBody = rt ? (rt.deleted ? "сообщение удалено" : rt.body) : "";
      }
      out.push({
        m, own: m.user_id === currentUserId, grouped,
        showDay: day !== prevDay, day, showName: showAuthorNames, replyAuthor, replyBody,
      });
      prevAuthor = m.user_id; prevDay = day; prevTime = t;
    }
    return out;
  }, [messages, currentUserId, showAuthorNames]);

  const onScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
    stickRef.current = nearBottom;
    setShowJump(!nearBottom);
    if (nearBottom) setNewCount(0);
  }, []);

  const jumpToBottom = useCallback(() => {
    stickRef.current = true;
    setShowJump(false);
    setNewCount(0);
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  // Автоскролл вниз при новом сообщении — только если пользователь у низа;
  // иначе копим счётчик новых для кнопки «вниз».
  useEffect(() => {
    const lastId = messages.length ? messages[messages.length - 1].id : 0;
    if (lastId === lastIdRef.current) return;
    const initial = lastIdRef.current === 0;
    const grew = lastId > lastIdRef.current;
    const prevLast = lastIdRef.current;
    lastIdRef.current = lastId;
    if (grew && (initial || stickRef.current)) {
      bottomRef.current?.scrollIntoView({ behavior: initial ? "auto" : "smooth" });
    } else if (grew && !initial) {
      const added = messages.filter((x) => x.id > prevLast).length;
      if (added > 0) setNewCount((c) => c + added);
    }
  }, [messages]);

  // Смена комнаты — сбрасываем состояние скролла/ввода/прочтения.
  useEffect(() => {
    lastIdRef.current = 0;
    stickRef.current = true;
    readSentRef.current = 0;
    setText(""); setMentionQuery(null); setShowJump(false); setEditing(null); setReplyTo(null); setReactPickerFor(null);
    onPeerTyping?.(false);
  }, [resetKey]); // eslint-disable-line react-hooks/exhaustive-deps

  // Автовысота поля ввода.
  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }, [text]);

  const onChange = (v: string) => {
    setText(v);
    // «печатает…»: троттлим отправку до раза в 2с.
    if (showTyping && roomId != null && v.trim()) {
      const now = Date.now();
      if (now - typingSentRef.current > 2000) {
        typingSentRef.current = now;
        sendTyping(roomId);
      }
    }
    if (members.length === 0) return; // упоминания не нужны (напр. в ЛС)
    const caret = inputRef.current?.selectionStart ?? v.length;
    const m = v.slice(0, caret).match(/@([\wА-Яа-яЁё]*)$/);
    setMentionQuery(m ? m[1].toLowerCase() : null);
  };

  const mentionMatches = useMemo(() => {
    if (mentionQuery == null) return [];
    // Показываем всех подходящих участников (список прокручивается), а не первые 6.
    return members.filter((m) => m.display_name.toLowerCase().includes(mentionQuery)).slice(0, 50);
  }, [mentionQuery, members]);

  const applyMention = (name: string) => {
    const first = name.split(" ")[0];
    setText((t) => t.replace(/@([\wА-Яа-яЁё]*)$/, `@${first} `));
    setMentionQuery(null);
    inputRef.current?.focus();
  };

  const submit = async () => {
    const body = text.trim();
    if (!body || sending || archived) return;
    setSending(true);
    try {
      if (editing) {
        // Сохраняем правку своего сообщения (текст обновится по SSE chat_edited).
        await editChatMessage(editing.id, body);
        setEditing(null);
        setText("");
        setMentionQuery(null);
      } else {
        stickRef.current = true;
        await send(body, replyTo?.id);
        setText("");
        setMentionQuery(null);
        setReplyTo(null);
        requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
      }
      // Держим фокус на поле — на мобиле клавиатура не должна закрываться.
      inputRef.current?.focus();
    } finally {
      setSending(false);
    }
  };

  const cancelEdit = () => { setEditing(null); setText(""); };

  const onLoadOlder = useCallback(async () => {
    prevHeightRef.current = scrollRef.current?.scrollHeight ?? 0;
    await loadOlder();
  }, [loadOlder]);

  // После подгрузки старых — сохраняем визуальную позицию (без прыжка).
  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (el && prevHeightRef.current) {
      el.scrollTop += el.scrollHeight - prevHeightRef.current;
      prevHeightRef.current = 0;
    }
  }, [messages]);

  const onDelete = useCallback((id: number, own: boolean) => {
    if (!window.confirm("Удалить сообщение?")) return;
    // Своё — пользовательский роут; чужое (админ) — админский.
    const p = own
      ? deleteChatMessage(id)
      : api.delete(`/api/admin/chat/messages/${id}`);
    p.catch(() => { /* no-op */ });
  }, []);

  const onEdit = useCallback((m: ChatMessage) => {
    setEditing({ id: m.id });
    setText(m.body);
    setMentionQuery(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  const startRecording = useCallback(async () => {
    if (!sendVoice || recording) return;
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      const mr = new MediaRecorder(stream);
      chunksRef.current = [];
      mr.ondataavailable = (e) => { if (e.data && e.data.size) chunksRef.current.push(e.data); };
      mr.start();
      mediaRecRef.current = mr;
      recStartRef.current = Date.now();
      setRecElapsed(0);
      setRecording(true);
      recTimerRef.current = setInterval(() => setRecElapsed(Math.floor((Date.now() - recStartRef.current) / 1000)), 500);
    } catch {
      toast.error("Нет доступа к микрофону — разрешите его в браузере");
    }
  }, [sendVoice, recording]);

  const finishRecording = useCallback((sendIt: boolean) => {
    const mr = mediaRecRef.current;
    if (recTimerRef.current) { clearInterval(recTimerRef.current); recTimerRef.current = null; }
    setRecording(false);
    if (!mr) return;
    const dur = Math.round((Date.now() - recStartRef.current) / 1000);
    mr.onstop = () => {
      streamRef.current?.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
      if (sendIt && chunksRef.current.length && dur >= 1) {
        stickRef.current = true;
        const blob = new Blob(chunksRef.current, { type: mr.mimeType || "audio/webm" });
        sendVoice?.(blob, dur).catch(() => toast.error("Не удалось отправить голосовое"));
        requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" }));
      }
      chunksRef.current = [];
      mediaRecRef.current = null;
    };
    try { mr.stop(); } catch { /* уже остановлен */ }
  }, [sendVoice]);

  // Остановить микрофон при размонтировании (смена комнаты/уход со страницы).
  useEffect(() => () => {
    if (recTimerRef.current) clearInterval(recTimerRef.current);
    streamRef.current?.getTracks().forEach((t) => t.stop());
  }, []);

  const fileInputRef = useRef<HTMLInputElement>(null);
  const onPickPhoto = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    e.target.value = "";
    if (!f || !sendPhoto) return;
    if (f.size > 8 * 1024 * 1024) { toast.error("Фото слишком большое (макс 8 МБ)"); return; }
    stickRef.current = true;
    sendPhoto(f)
      .then(() => requestAnimationFrame(() => bottomRef.current?.scrollIntoView({ behavior: "smooth" })))
      .catch(() => toast.error("Не удалось отправить фото"));
  }, [sendPhoto]);

  return (
    <div className="flex flex-col h-full min-h-0">
      {/* Сообщения. Внутренний spacer flex-1 прижимает переписку к низу. */}
      <div className="relative flex-1 min-h-0">
        <div ref={scrollRef} onScroll={onScroll} className="chat-surface h-full overflow-y-auto overscroll-contain">
          <div className="flex min-h-full flex-col px-3 py-3">
            <div className="flex-1" />
            {hasMore && (
              <div className="text-center pb-3">
                <button onClick={onLoadOlder} className="chat-pill rounded-full px-3.5 py-1.5 text-[11px] font-medium text-zinc-300 hover:text-white transition-colors">
                  Показать ранние
                </button>
              </div>
            )}
            {rows.length === 0 ? (
              <div className="flex flex-col items-center gap-2 py-14 text-center">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-yellow-400/10 text-yellow-400">
                  <Send size={20} />
                </div>
                <p className="text-sm font-medium text-zinc-400">Пока нет сообщений</p>
                <p className="text-xs text-zinc-600">Напишите первым — начните общение</p>
              </div>
            ) : (
              rows.map((row) => (
                <MessageRow key={row.m.id} {...row} isAdmin={isAdmin} onDelete={onDelete} onEdit={onEdit}
                  onReply={onReply} onReact={onReact} onPickEmoji={onPickEmoji} onToggleReaction={toggleReaction}
                  onImageClick={onImageClick} onReplyClick={onReplyClick} onLongPress={onLongPress} onQuickLike={onQuickLike}
                  showReceipts={showReceipts} otherReads={otherReads}
                  reactions={reactionsByMsg[row.m.id] ?? EMPTY_REACTIONS} pickerOpen={reactPickerFor === row.m.id}
                  showUnread={row.m.id === firstUnreadId} />
              ))
            )}
            <div ref={bottomRef} />
          </div>
        </div>
        {showJump && (
          <button
            onClick={jumpToBottom}
            aria-label="Вниз к последним сообщениям"
            className="chat-pill absolute bottom-3 right-3 flex h-10 min-w-[2.5rem] items-center justify-center gap-1 rounded-full px-2 text-zinc-200 hover:text-yellow-400 transition-colors"
          >
            {newCount > 0 && <span className="text-xs font-bold text-yellow-400">{newCount > 99 ? "99+" : newCount}</span>}
            <ChevronDown size={19} />
          </button>
        )}
      </div>

      {lightboxUrl && <Lightbox url={lightboxUrl} onClose={() => setLightboxUrl(null)} />}

      {/* Меню долгого нажатия (мобайл) */}
      {menuFor && (
        <div className="fixed inset-0 z-[65] flex items-end" onClick={() => setMenuFor(null)}>
          <div className="absolute inset-0 bg-black/50" />
          <div
            className="relative w-full rounded-t-2xl border-t border-white/10 bg-zinc-900 p-3 pb-[max(0.9rem,env(safe-area-inset-bottom))]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mx-auto mb-3 h-1 w-10 rounded-full bg-zinc-700" />
            <div className="mb-2 flex justify-around">
              {REACTION_EMOJIS.map((e) => (
                <button key={e} onClick={() => { onPickEmoji(menuFor.id, e); setMenuFor(null); }} className="text-2xl transition-transform hover:scale-125" aria-label={`Реакция ${e}`}>
                  {e}
                </button>
              ))}
            </div>
            <div className="space-y-0.5">
              <button onClick={() => { onReply(menuFor); setMenuFor(null); }} className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-zinc-200 hover:bg-white/5">
                <Reply size={17} className="text-zinc-400" /> Ответить
              </button>
              {menuFor.body && (
                <button onClick={() => { navigator.clipboard?.writeText(menuFor.body).catch(() => {}); setMenuFor(null); }} className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-zinc-200 hover:bg-white/5">
                  <Copy size={17} className="text-zinc-400" /> Копировать
                </button>
              )}
              {menuFor.user_id === currentUserId && menuFor.body && (
                <button onClick={() => { onEdit(menuFor); setMenuFor(null); }} className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-zinc-200 hover:bg-white/5">
                  <Pencil size={17} className="text-zinc-400" /> Изменить
                </button>
              )}
              {(menuFor.user_id === currentUserId || isAdmin) && (
                <button onClick={() => { onDelete(menuFor.id, menuFor.user_id === currentUserId); setMenuFor(null); }} className="flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm text-red-400 hover:bg-red-500/10">
                  <Trash2 size={17} /> Удалить
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Ввод */}
      {archived ? (
        <div className="border-t border-zinc-800 px-4 py-3 text-center text-xs text-zinc-500 flex-shrink-0">
          {archivedNote}
        </div>
      ) : (
        <div className="relative border-t border-white/5 bg-zinc-950/30 p-2 flex-shrink-0 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
          {editing && (
            <div className="mb-1.5 flex items-center gap-2 rounded-xl border-l-2 border-yellow-400/70 bg-zinc-800/50 px-3 py-1.5">
              <Pencil size={13} className="flex-shrink-0 text-yellow-400" />
              <span className="flex-1 text-xs text-zinc-400">Редактирование сообщения</span>
              <button onClick={cancelEdit} aria-label="Отменить" className="rounded p-0.5 text-zinc-500 hover:text-zinc-200">
                <X size={14} />
              </button>
            </div>
          )}
          {replyTo && !editing && (
            <div className="mb-1.5 flex items-center gap-2 rounded-xl border-l-2 border-yellow-400/70 bg-zinc-800/50 px-3 py-1.5">
              <Reply size={13} className="flex-shrink-0 text-yellow-400" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-[11px] font-semibold text-yellow-300/90">Ответ · {replyTo.author}</p>
                <p className="truncate text-[11px] text-zinc-400">{replyTo.body}</p>
              </div>
              <button onClick={() => setReplyTo(null)} aria-label="Отменить ответ" className="rounded p-0.5 text-zinc-500 hover:text-zinc-200">
                <X size={14} />
              </button>
            </div>
          )}
          {mentionMatches.length > 0 && (
            <div className="absolute bottom-full left-2 right-2 mb-2 max-h-52 overflow-y-auto rounded-2xl border border-white/10 bg-zinc-900/95 backdrop-blur shadow-2xl shadow-black/50">
              {mentionMatches.map((m) => (
                <button
                  key={m.user_id}
                  onClick={() => applyMention(m.display_name)}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm text-zinc-200 hover:bg-white/5 transition-colors"
                >
                  <PlayerAvatar displayName={m.display_name} favoriteClub={m.favorite_club} size={22} />
                  <span className="truncate">{m.display_name}</span>
                </button>
              ))}
            </div>
          )}
          {recording ? (
            /* Идёт запись голосового */
            <div className="flex items-center gap-3 rounded-[1.4rem] border border-red-500/40 bg-zinc-950/70 px-4 py-2.5">
              <span className="h-3 w-3 flex-shrink-0 animate-pulse rounded-full bg-red-500" />
              <span className="tabular-nums text-sm font-semibold text-zinc-200">{fmtDur(recElapsed)}</span>
              <span className="flex-1 truncate text-xs text-zinc-500">Идёт запись…</span>
              <button
                onClick={() => finishRecording(false)}
                aria-label="Отменить запись"
                className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full text-zinc-400 hover:text-red-400"
              >
                <Trash2 size={18} />
              </button>
              <button
                onClick={() => finishRecording(true)}
                aria-label="Отправить голосовое"
                className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-[#d9ff3d] to-[#a3cc1e] text-zinc-950 shadow-[0_3px_14px_rgba(200,241,53,0.35)] active:scale-90"
              >
                <Send size={17} />
              </button>
            </div>
          ) : (
            <div className="flex items-end gap-2">
              <div className="flex flex-1 items-end gap-1.5 rounded-[1.4rem] border border-zinc-700/70 bg-zinc-950/70 px-2 py-1 transition-colors focus-within:border-yellow-400/70 focus-within:ring-2 focus-within:ring-yellow-400/15">
                {sendPhoto && (
                  <button
                    onClick={() => fileInputRef.current?.click()}
                    onMouseDown={(e) => e.preventDefault()}
                    aria-label="Прикрепить фото"
                    className="mb-1 flex-shrink-0 rounded-full p-1.5 text-zinc-500 hover:text-yellow-400 transition-colors"
                  >
                    <ImagePlus size={20} />
                  </button>
                )}
                <textarea
                  ref={inputRef}
                  rows={1}
                  value={text}
                  onChange={(e) => onChange(e.target.value)}
                  onKeyDown={(e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); submit(); } }}
                  placeholder={placeholder}
                  maxLength={2000}
                  className="flex-1 resize-none border-0 bg-transparent px-1 py-1.5 text-[15px] text-zinc-100 placeholder-zinc-600 focus:outline-none leading-snug max-h-[120px]"
                />
              </div>
              <input ref={fileInputRef} type="file" accept="image/*" className="hidden" onChange={onPickPhoto} />
              {text.trim() || editing || !sendVoice ? (
                <button
                  onClick={submit}
                  onMouseDown={(e) => e.preventDefault()} /* не даём кнопке забрать фокус — клавиатура не закрывается */
                  disabled={sending || !text.trim()}
                  className={cn(
                    "flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full text-zinc-950 transition-all active:scale-90",
                    "bg-gradient-to-br from-[#d9ff3d] to-[#a3cc1e]",
                    text.trim() ? "shadow-[0_3px_14px_rgba(200,241,53,0.35)]" : "opacity-40",
                  )}
                  aria-label={editing ? "Сохранить" : "Отправить"}
                >
                  {editing ? <Check size={18} strokeWidth={2.5} /> : <Send size={17} />}
                </button>
              ) : (
                <button
                  onClick={startRecording}
                  aria-label="Записать голосовое"
                  className="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-full bg-zinc-800 text-zinc-200 transition-all hover:bg-zinc-700 active:scale-90"
                >
                  <Mic size={18} />
                </button>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
