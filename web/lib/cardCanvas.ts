import { getActiveLang } from "@/lib/i18n";
import { getClub } from "@/lib/clubs";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "";

export interface CardInput {
  name: string;
  rank: string;
  rating: number;
  favoriteClub?: string;
  wins: number;
  draws: number;
  losses: number;
  goals: number;
  winRate: number;
}

type RGB = { r: number; g: number; b: number };

// ── Цвета (порт логики из серверного cardgen) ────────────────────────────────
function hex(s: string): RGB {
  const c = s.replace("#", "");
  if (c.length < 6) return { r: 120, g: 120, b: 130 };
  return { r: parseInt(c.slice(0, 2), 16), g: parseInt(c.slice(2, 4), 16), b: parseInt(c.slice(4, 6), 16) };
}
function lum(c: RGB) { return 0.299 * c.r + 0.587 * c.g + 0.114 * c.b; }
function rgba(c: RGB, a = 1) { return `rgba(${c.r},${c.g},${c.b},${a})`; }
function darken(c: RGB, f: number): RGB { return { r: (c.r * f) | 0, g: (c.g * f) | 0, b: (c.b * f) | 0 }; }
function dist(a: RGB, b: RGB) { const dr = a.r - b.r, dg = a.g - b.g, db = a.b - b.b; return dr * dr + dg * dg + db * db; }
function tierAccent(rating: number): RGB {
  if (rating >= 1500) return { r: 251, g: 191, b: 36 };
  if (rating >= 1300) return { r: 96, g: 165, b: 250 };
  if (rating >= 1150) return { r: 167, g: 139, b: 250 };
  return { r: 74, g: 222, b: 128 };
}
function accentColor(club: ReturnType<typeof getClub>, rating: number): RGB {
  if (club) {
    const c1 = hex(club.color); if (lum(c1) >= 45) return c1;
    const c2 = hex(club.color2); if (lum(c2) >= 45) return c2;
  }
  return tierAccent(rating);
}
function secondColor(club: ReturnType<typeof getClub>, accent: RGB): RGB {
  if (club) {
    const c2 = hex(club.color2); if (dist(c2, accent) > 60) return c2;
    const c1 = hex(club.color); if (dist(c1, accent) > 60) return c1;
  }
  return darken(accent, 0.55);
}
function contrastText(c: RGB) { return lum(c) > 150 ? "#0f1219" : "#ffffff"; }

function starsFor(r: number) {
  if (r >= 1250) return 5;
  if (r >= 1150) return 4;
  if (r >= 1050) return 3;
  if (r >= 950) return 2;
  return 1;
}
const isLetter = (ch: string) => ch.toLowerCase() !== ch.toUpperCase();
function cleanRank(s: string) {
  const arr = Array.from(s);
  let i = 0;
  while (i < arr.length && !isLetter(arr[i])) i++;
  return arr.slice(i).join("").trim().toUpperCase();
}
function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return "?";
  let out = Array.from(parts[0])[0] || "";
  if (parts.length > 1) out += Array.from(parts[1])[0] || "";
  return out.toUpperCase();
}
function truncate(s: string, max: number) {
  const a = Array.from(s);
  return a.length <= max ? s : a.slice(0, max - 1).join("") + "…";
}

function loadImage(src: string): Promise<HTMLImageElement | null> {
  return new Promise((res) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => res(img);
    img.onerror = () => res(null);
    img.src = src;
  });
}

/** Рисует звезду пятиконечную. */
function star(ctx: CanvasRenderingContext2D, cx: number, cy: number, r: number, fill: string) {
  ctx.beginPath();
  for (let i = 0; i < 10; i++) {
    const ang = -Math.PI / 2 + (i * Math.PI) / 5;
    const rad = i % 2 === 1 ? r * 0.42 : r;
    const x = cx + Math.cos(ang) * rad;
    const y = cy + Math.sin(ang) * rad;
    if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
  }
  ctx.closePath();
  ctx.fillStyle = fill;
  ctx.fill();
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

/**
 * Рисует карточку игрока на canvas. Тот же дизайн, что и серверный, но в
 * кружке — настоящий логотип клуба (через CORS-прокси), флаг сборной или
 * монограмма (если клуба нет).
 */
export async function renderCard(canvas: HTMLCanvasElement, d: CardInput): Promise<void> {
  const s = 2;
  const W = 600 * s, H = 350 * s;
  canvas.width = W;
  canvas.height = H;
  const ctx = canvas.getContext("2d")!;
  ctx.clearRect(0, 0, W, H);

  const club = getClub(d.favoriteClub);
  const accent = accentColor(club, d.rating);
  const accent2 = secondColor(club, accent);
  const A = rgba(accent), A2 = rgba(accent2);
  const F = "Arial, sans-serif";

  // фон
  const bg = ctx.createLinearGradient(0, 0, W, H);
  bg.addColorStop(0, "rgb(24,28,41)");
  bg.addColorStop(1, "rgb(7,10,17)");
  ctx.fillStyle = bg; ctx.fillRect(0, 0, W, H);

  // свечение
  const glow = ctx.createRadialGradient(s * 120, s * 64, 0, s * 120, s * 64, s * 330);
  glow.addColorStop(0, rgba(accent, 0.26));
  glow.addColorStop(1, rgba(accent, 0));
  ctx.fillStyle = glow; ctx.fillRect(0, 0, W, H);

  // диагональный блик
  ctx.fillStyle = "rgba(255,255,255,0.028)";
  ctx.beginPath();
  ctx.moveTo(W * 0.56, 0); ctx.lineTo(W * 0.70, 0); ctx.lineTo(W * 0.40, H); ctx.lineTo(W * 0.26, H);
  ctx.closePath(); ctx.fill();

  // виньетка
  const vig = ctx.createRadialGradient(W / 2, H / 2, H * 0.35, W / 2, H / 2, W * 0.72);
  vig.addColorStop(0, "rgba(0,0,0,0)");
  vig.addColorStop(1, "rgba(0,0,0,0.37)");
  ctx.fillStyle = vig; ctx.fillRect(0, 0, W, H);

  // декоративная инициала
  ctx.font = `900 ${s * 200}px ${F}`;
  ctx.fillStyle = rgba(accent, 0.06);
  ctx.textAlign = "center"; ctx.textBaseline = "middle";
  ctx.fillText(initials(d.name), s * 480, s * 220);

  // ── бейдж клуба ──
  const cx = s * 106, cy = s * 150, rr = s * 68;

  // кольцо
  const ring = ctx.createLinearGradient(cx - rr, cy - rr, cx + rr, cy + rr);
  ring.addColorStop(0, A); ring.addColorStop(1, A2);
  ctx.fillStyle = ring;
  ctx.beginPath(); ctx.arc(cx, cy, rr + s * 5, 0, Math.PI * 2); ctx.fill();

  const logoImg = club && club.logoUrl && !club.isNational
    ? await loadImage(`${API_URL}/api/club-logo?club=${club.id}`)
    : null;

  if (logoImg) {
    // светлый холдер + логотип
    ctx.fillStyle = "rgb(244,245,248)";
    ctx.beginPath(); ctx.arc(cx, cy, rr, 0, Math.PI * 2); ctx.fill();
    const box = rr * 1.42;
    const iw = logoImg.width, ih = logoImg.height;
    const sc = box / Math.max(iw, ih);
    const dw = iw * sc, dh = ih * sc;
    ctx.drawImage(logoImg, cx - dw / 2, cy - dh / 2, dw, dh);
  } else {
    // градиентный кружок
    ctx.save();
    ctx.beginPath(); ctx.arc(cx, cy, rr, 0, Math.PI * 2); ctx.clip();
    const core = ctx.createLinearGradient(cx - rr, cy - rr, cx + rr, cy + rr);
    core.addColorStop(0, A); core.addColorStop(1, A2);
    ctx.fillStyle = core; ctx.fillRect(cx - rr, cy - rr, 2 * rr, 2 * rr);
    // глянец
    const gloss = ctx.createLinearGradient(cx, cy - rr, cx, cy + s * 8);
    gloss.addColorStop(0, "rgba(255,255,255,0.27)");
    gloss.addColorStop(1, "rgba(255,255,255,0)");
    ctx.fillStyle = gloss; ctx.fillRect(cx - rr, cy - rr, 2 * rr, rr + s * 8);
    ctx.restore();
    // содержимое: флаг сборной или монограмма
    if (club && club.isNational) {
      ctx.font = `${rr * 1.15}px ${F}`;
      ctx.textAlign = "center"; ctx.textBaseline = "middle";
      ctx.fillText(club.logo, cx, cy + s * 2);
    } else {
      ctx.font = `900 ${s * 48}px ${F}`;
      ctx.fillStyle = contrastText(accent);
      ctx.textAlign = "center"; ctx.textBaseline = "middle";
      ctx.fillText(initials(d.name), cx, cy);
    }
  }

  // внутреннее кольцо
  ctx.strokeStyle = "rgba(255,255,255,0.18)";
  ctx.lineWidth = s * 1.5;
  ctx.beginPath(); ctx.arc(cx, cy, rr - s * 6, 0, Math.PI * 2); ctx.stroke();

  // название клуба
  if (club) {
    ctx.font = `bold ${s * 15}px ${F}`;
    ctx.fillStyle = "rgba(255,255,255,0.92)";
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(truncate(getActiveLang() === "ru" ? club.nameRu || club.name : club.name, 18), cx, cy + rr + s * 30);
  }

  // ── правая колонка ──
  const rx = s * 206;

  // бейдж ранга
  const rank = cleanRank(d.rank);
  if (rank) {
    ctx.font = `bold ${s * 12}px ${F}`;
    const tw = ctx.measureText(rank).width;
    const padX = s * 11, ph = s * 24, pw2 = tw + 2 * padX, px = rx, py2 = s * 34;
    ctx.fillStyle = rgba(accent, 0.13);
    roundRect(ctx, px, py2, pw2, ph, ph / 2); ctx.fill();
    ctx.strokeStyle = A; ctx.lineWidth = s * 1.4;
    roundRect(ctx, px, py2, pw2, ph, ph / 2); ctx.stroke();
    ctx.fillStyle = A; ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.fillText(rank, px + pw2 / 2, py2 + ph / 2 + s * 1);
  }

  // имя
  ctx.font = `900 ${s * 30}px ${F}`;
  ctx.fillStyle = "rgba(255,255,255,0.98)";
  ctx.textAlign = "left"; ctx.textBaseline = "alphabetic";
  ctx.fillText(truncate(d.name, 18), rx, s * 96);

  // ELO
  const elo = String(d.rating);
  ctx.font = `900 ${s * 56}px ${F}`;
  ctx.fillStyle = A;
  ctx.fillText(elo, rx, s * 158);
  const ew = ctx.measureText(elo).width;
  ctx.font = `${s * 15}px ${F}`;
  ctx.fillStyle = "rgba(255,255,255,0.4)";
  ctx.fillText("ELO", rx + ew + s * 12, s * 152);

  // звёзды
  const stars = starsFor(d.rating);
  const starR = s * 8.5;
  let starX = rx + starR;
  for (let i = 0; i < 5; i++) {
    star(ctx, starX, s * 180, starR, i < stars ? A : "rgba(255,255,255,0.11)");
    starX += starR * 2 + s * 7;
  }

  // панель статистики
  const py = s * 208, pw = W - rx - s * 22;
  ctx.fillStyle = "rgba(255,255,255,0.04)";
  roundRect(ctx, rx, py, pw, s * 92, s * 14); ctx.fill();
  ctx.strokeStyle = "rgba(255,255,255,0.08)"; ctx.lineWidth = s * 1;
  roundRect(ctx, rx, py, pw, s * 92, s * 14); ctx.stroke();

  const sy = py + s * 46, step = pw / 5, cx0 = rx + step / 2;
  const statList: [string, string, string][] = [
    [String(d.wins), "W", "rgb(34,197,94)"],
    [String(d.draws), "D", "rgb(234,179,8)"],
    [String(d.losses), "L", "rgb(239,68,68)"],
    [`${Math.round(d.winRate)}%`, "WIN", "rgb(129,140,248)"],
    [String(d.goals), "GOALS", "rgb(251,146,60)"],
  ];
  statList.forEach(([val, label, col], i) => {
    const x = cx0 + i * step;
    ctx.textAlign = "center"; ctx.textBaseline = "middle";
    ctx.font = `900 ${s * 20}px ${F}`; ctx.fillStyle = col;
    ctx.fillText(val, x, sy);
    ctx.font = `${s * 10}px ${F}`; ctx.fillStyle = "rgba(255,255,255,0.42)";
    ctx.fillText(label, x, sy + s * 18);
  });

  // рамка
  ctx.strokeStyle = rgba(accent, 0.37); ctx.lineWidth = s * 1.5;
  roundRect(ctx, s * 8, s * 8, W - s * 16, H - s * 16, s * 18); ctx.stroke();

  // нижняя полоса
  const stripe = ctx.createLinearGradient(0, 0, W, 0);
  stripe.addColorStop(0, A); stripe.addColorStop(1, A2);
  ctx.fillStyle = stripe; ctx.fillRect(0, H - s * 5, W, s * 5);

  // футер
  ctx.font = `bold ${s * 12}px ${F}`;
  ctx.fillStyle = "rgba(255,255,255,0.32)";
  ctx.textAlign = "right"; ctx.textBaseline = "middle";
  ctx.fillText("eFootLeague", W - s * 22, H - s * 24);
}

/** Экспортирует canvas в PNG-Blob. */
export function canvasToBlob(canvas: HTMLCanvasElement): Promise<Blob | null> {
  return new Promise((res) => canvas.toBlob((b) => res(b), "image/png"));
}
