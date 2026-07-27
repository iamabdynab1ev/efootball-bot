"use client";

import { toast } from "sonner";
import { getClub } from "@/lib/clubs";

// Шеринг-карточки: результат матча и трофей рисуются на canvas прямо в
// браузере (ноль нагрузки на сервер) и уходят в системный шеринг картинкой —
// готовый пост для любого чата. Стиль единый с карточкой игрока (cardCanvas).

const W = 1000, H = 560;

function loadImage(src: string): Promise<HTMLImageElement | null> {
  return new Promise((res) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => res(img);
    img.onerror = () => res(null);
    img.src = src;
  });
}

function roundRect(ctx: CanvasRenderingContext2D, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

function initials(name: string) {
  const p = name.trim().split(/\s+/).filter(Boolean);
  return ((Array.from(p[0] ?? "?")[0] ?? "?") + (p[1] ? Array.from(p[1])[0] : "")).toUpperCase();
}

function truncate(s: string, max: number) {
  const a = Array.from(s);
  return a.length <= max ? s : a.slice(0, max - 1).join("") + "…";
}

// Ночной стадион: небо, свет прожекторов, газон с разметкой — фон обеих карточек.
function drawStadium(ctx: CanvasRenderingContext2D) {
  const sky = ctx.createRadialGradient(W / 2, -80, 60, W / 2, 0, H * 1.25);
  sky.addColorStop(0, "#16233b");
  sky.addColorStop(0.5, "#0a111d");
  sky.addColorStop(1, "#05070b");
  ctx.fillStyle = sky;
  ctx.fillRect(0, 0, W, H);

  // Прожекторный свет сверху
  const beam = ctx.createRadialGradient(W / 2, -40, 10, W / 2, -40, 520);
  beam.addColorStop(0, "rgba(255,255,255,0.14)");
  beam.addColorStop(1, "rgba(255,255,255,0)");
  ctx.fillStyle = beam;
  ctx.fillRect(0, 0, W, H);

  // Газон снизу с полосами
  const grassTop = H * 0.62;
  const grass = ctx.createLinearGradient(0, grassTop, 0, H);
  grass.addColorStop(0, "rgba(16,49,26,0)");
  grass.addColorStop(1, "rgba(16,49,26,0.55)");
  ctx.fillStyle = grass;
  ctx.fillRect(0, grassTop, W, H - grassTop);
  ctx.fillStyle = "rgba(74,222,128,0.05)";
  for (let x = 0; x < W; x += 160) ctx.fillRect(x, grassTop, 80, H - grassTop);

  // Центральный круг разметки
  ctx.strokeStyle = "rgba(255,255,255,0.08)";
  ctx.lineWidth = 3;
  ctx.beginPath();
  ctx.arc(W / 2, H + 60, 240, Math.PI, 2 * Math.PI);
  ctx.stroke();
}

// Брендинг-подвал: eFootLeague + вольтовая точка.
function drawBrand(ctx: CanvasRenderingContext2D, sub: string) {
  ctx.textAlign = "center";
  ctx.fillStyle = "#c8f135";
  ctx.font = "900 30px Unbounded, Inter, system-ui, sans-serif";
  ctx.fillText("eFootLeague", W / 2, H - 36);
  ctx.fillStyle = "rgba(228,228,231,0.45)";
  ctx.font = "600 20px Inter, system-ui, sans-serif";
  ctx.fillText(sub, W / 2, H - 10);
}

async function drawSide(ctx: CanvasRenderingContext2D, x: number, name: string, clubID: string | undefined) {
  const club = getClub(clubID);
  const img = club?.logoUrl && !club.isNational ? await loadImage(club.logoUrl) : null;
  const cy = 250;
  if (img) {
    ctx.save();
    ctx.shadowColor = "rgba(0,0,0,0.6)";
    ctx.shadowBlur = 26;
    ctx.drawImage(img, x - 70, cy - 70, 140, 140);
    ctx.restore();
  } else {
    // Кружок с инициалами
    ctx.fillStyle = "#27272a";
    ctx.beginPath(); ctx.arc(x, cy, 68, 0, Math.PI * 2); ctx.fill();
    ctx.strokeStyle = "rgba(250,204,21,0.5)";
    ctx.lineWidth = 4; ctx.stroke();
    ctx.fillStyle = "#fafafa";
    ctx.font = "900 52px Unbounded, Inter, system-ui, sans-serif";
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(initials(name), x, cy + 4);
    ctx.textBaseline = "alphabetic";
  }
  ctx.textAlign = "center";
  ctx.fillStyle = "#fafafa";
  ctx.font = "800 34px Inter, system-ui, sans-serif";
  ctx.fillText(truncate(name, 14), x, cy + 122);
}

export interface MatchCardData {
  homeName: string;
  awayName: string;
  homeClub?: string;
  awayClub?: string;
  homeGoals: number;
  awayGoals: number;
  context: string; // «Лига Чемпионов ⚽ · Тур 3» или «Товарищеский матч»
}

async function renderMatchCard(canvas: HTMLCanvasElement, d: MatchCardData) {
  canvas.width = W; canvas.height = H;
  const ctx = canvas.getContext("2d")!;
  try { await (document as Document & { fonts?: FontFaceSet }).fonts?.ready; } catch { /* ignore */ }

  drawStadium(ctx);

  // Заголовок-контекст
  ctx.textAlign = "center";
  ctx.fillStyle = "#facc15";
  ctx.font = "800 24px Inter, system-ui, sans-serif";
  ctx.fillText(truncate(d.context, 40).toUpperCase(), W / 2, 78);
  ctx.fillStyle = "rgba(228,228,231,0.6)";
  ctx.font = "700 20px Inter, system-ui, sans-serif";
  ctx.fillText("ФИНАЛЬНЫЙ СЧЁТ", W / 2, 110);

  // Стороны
  await drawSide(ctx, 190, d.homeName, d.homeClub);
  await drawSide(ctx, W - 190, d.awayName, d.awayClub);

  // Счёт по центру — самое крупное на карточке
  ctx.textAlign = "center";
  ctx.save();
  ctx.shadowColor = "rgba(200,241,53,0.35)";
  ctx.shadowBlur = 40;
  ctx.fillStyle = "#fafafa";
  ctx.font = "900 150px Unbounded, Inter, system-ui, sans-serif";
  ctx.fillText(`${d.homeGoals}:${d.awayGoals}`, W / 2, 305);
  ctx.restore();

  drawBrand(ctx, "Лиги · Рейтинг · Трофеи");
}

export interface TrophyCardData {
  emoji: string;
  label: string;      // «Чемпион»
  playerName: string;
  context?: string;   // лига
}

async function renderTrophyCard(canvas: HTMLCanvasElement, d: TrophyCardData) {
  canvas.width = W; canvas.height = H;
  const ctx = canvas.getContext("2d")!;
  try { await (document as Document & { fonts?: FontFaceSet }).fonts?.ready; } catch { /* ignore */ }

  drawStadium(ctx);

  // Золотые лучи из центра медали
  ctx.save();
  ctx.translate(W / 2, 210);
  for (let i = 0; i < 18; i++) {
    ctx.rotate(Math.PI / 9);
    const ray = ctx.createLinearGradient(0, 0, 0, -320);
    ray.addColorStop(0, "rgba(250,204,21,0.16)");
    ray.addColorStop(1, "rgba(250,204,21,0)");
    ctx.fillStyle = ray;
    ctx.beginPath();
    ctx.moveTo(0, 0); ctx.lineTo(-26, -320); ctx.lineTo(26, -320);
    ctx.closePath(); ctx.fill();
  }
  ctx.restore();

  // Медаль
  const medal = ctx.createRadialGradient(W / 2 - 30, 170, 20, W / 2, 210, 120);
  medal.addColorStop(0, "#fde68a");
  medal.addColorStop(0.55, "#f59e0b");
  medal.addColorStop(1, "#a16207");
  ctx.save();
  ctx.shadowColor = "rgba(250,204,21,0.5)";
  ctx.shadowBlur = 60;
  ctx.fillStyle = medal;
  ctx.beginPath(); ctx.arc(W / 2, 210, 110, 0, Math.PI * 2); ctx.fill();
  ctx.restore();
  ctx.strokeStyle = "rgba(253,230,138,0.8)";
  ctx.lineWidth = 6;
  ctx.beginPath(); ctx.arc(W / 2, 210, 110, 0, Math.PI * 2); ctx.stroke();
  ctx.textAlign = "center"; ctx.textBaseline = "middle";
  ctx.font = "96px system-ui, sans-serif";
  ctx.fillText(d.emoji || "🏆", W / 2, 218);
  ctx.textBaseline = "alphabetic";

  // Тексты
  ctx.fillStyle = "#facc15";
  ctx.font = "800 24px Inter, system-ui, sans-serif";
  ctx.fillText("НОВЫЙ ТРОФЕЙ", W / 2, 372);
  ctx.fillStyle = "#fafafa";
  ctx.font = "900 58px Unbounded, Inter, system-ui, sans-serif";
  ctx.fillText(truncate(d.label, 20), W / 2, 428);
  ctx.fillStyle = "rgba(228,228,231,0.75)";
  ctx.font = "700 26px Inter, system-ui, sans-serif";
  ctx.fillText(truncate(`${d.playerName}${d.context ? " · " + d.context : ""}`, 44), W / 2, 466);

  drawBrand(ctx, "Играй в турниры по eFootball");
}

// Системный шеринг картинкой; без поддержки — скачивание файла.
async function shareCanvas(canvas: HTMLCanvasElement, filename: string, text: string) {
  const blob = await new Promise<Blob | null>((res) => canvas.toBlob(res, "image/png"));
  if (!blob) throw new Error("no blob");
  const file = new File([blob], filename, { type: "image/png" });
  const navAny = navigator as Navigator & { canShare?: (d: ShareData) => boolean };
  if (navAny.canShare && navAny.canShare({ files: [file] })) {
    await navigator.share({ files: [file], title: "eFootLeague", text });
  } else {
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
    toast.success("Картинка сохранена — отправь её в чат");
  }
}

export async function shareMatchCard(d: MatchCardData) {
  const c = document.createElement("canvas");
  await renderMatchCard(c, d);
  await shareCanvas(c, "efootleague-match.png", `⚽ ${d.homeName} ${d.homeGoals}:${d.awayGoals} ${d.awayName}`);
}

export async function shareTrophyCard(d: TrophyCardData) {
  const c = document.createElement("canvas");
  await renderTrophyCard(c, d);
  await shareCanvas(c, "efootleague-trophy.png", `🏆 ${d.playerName} — «${d.label}» в eFootLeague!`);
}
