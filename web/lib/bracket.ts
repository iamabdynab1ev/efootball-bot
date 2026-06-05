import { BracketSlot, BracketStage } from "@/lib/api";

const STAGE_META: Record<number, { key: string; label: string }> = {
  1:  { key: "final", label: "Финал"        },
  2:  { key: "sf",    label: "Полуфинал"    },
  4:  { key: "qf",    label: "Четвертьфинал"},
  8:  { key: "r16",   label: "1/8 финала"   },
  16: { key: "r32",   label: "1/16 финала"  },
};

function emptySlot(stage: string, slot: number, homeName = "", awayName = ""): BracketSlot {
  return { slot, stage, home_name: homeName, away_name: awayName };
}

/** Количество участников плей-офф из настроек лиги. */
export function playoffSize(numGroups: number, groupAdvance: number): number {
  return numGroups * groupAdvance;
}

/**
 * Кросс-групповой посев для 2 групп:
 * 1A vs 4B, 2A vs 3B, 1B vs 4A, 2B vs 3A
 * Возвращает массив { home, away } пар в порядке слотов.
 */
function crossSeedLabels(numGroups: number, advance: number): { home: string; away: string }[] {
  if (numGroups !== 2) {
    // Для 3+ групп: последовательно
    const pairs: { home: string; away: string }[] = [];
    const letters = Array.from({ length: numGroups }, (_, i) => String.fromCharCode(65 + i));
    let advancing: string[] = [];
    for (const g of letters) {
      for (let i = 1; i <= advance; i++) advancing.push(`${i}${g}`);
    }
    for (let i = 0; i < advancing.length - 1; i += 2) {
      pairs.push({ home: advancing[i], away: advancing[i + 1] });
    }
    return pairs;
  }

  const n = advance;
  const half = Math.floor(n / 2);
  const pairs: { home: string; away: string }[] = [];

  // Первая половина: A[0..half-1] vs B[n-1..n-half]
  for (let i = 0; i < half; i++) {
    pairs.push({ home: `${i + 1}A`, away: `${n - i}B` });
  }
  // Вторая половина: B[0..half-1] vs A[n-1..n-half]
  for (let i = 0; i < half; i++) {
    pairs.push({ home: `${i + 1}B`, away: `${n - i}A` });
  }
  // Нечётный advance: средний A vs средний B
  if (n % 2 !== 0) {
    const mid = Math.floor(n / 2) + 1;
    pairs.push({ home: `${mid}A`, away: `${mid}B` });
  }
  return pairs;
}

/**
 * Строит пустую сетку с подписями посева для группового формата.
 * Для 2 групп с advance=4: QF показывает "1A", "4B" и т.д.
 */
export function buildSeededEmptyBracket(numGroups: number, advance: number): BracketStage[] {
  const total = numGroups * advance;
  if (total < 2) return [];

  let size = 1;
  while (size < total) size *= 2;

  const stages: BracketStage[] = [];
  let matches = size / 2;
  let isFirstRound = true;

  while (matches >= 1) {
    const meta = STAGE_META[matches] ?? {
      key: `r${matches * 2}`,
      label: `1/${matches * 2} финала`,
    };

    let slots: BracketSlot[];

    if (isFirstRound && numGroups >= 2) {
      const labels = crossSeedLabels(numGroups, advance);
      slots = Array.from({ length: matches }, (_, i) => {
        const lbl = labels[i];
        return emptySlot(meta.key, i + 1, lbl?.home ?? "", lbl?.away ?? "");
      });
    } else {
      slots = Array.from({ length: matches }, (_, i) =>
        emptySlot(meta.key, i + 1, "TBD", "TBD")
      );
    }

    stages.push({ stage: meta.key, label: meta.label, slots });
    matches = Math.floor(matches / 2);
    isFirstRound = false;
  }

  return stages;
}

/** Обычная пустая сетка без подписей (для кубка и т.п.) */
export function buildEmptyBracket(n: number): BracketStage[] {
  return buildSeededEmptyBracket(1, n);
}
