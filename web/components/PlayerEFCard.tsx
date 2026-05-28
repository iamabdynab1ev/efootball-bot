"use client";

import { Star } from "lucide-react";
import { cn } from "@/lib/utils";

interface Props {
  name: string;
  rating: number;
  rank?: string;
  teamPower?: number;
  wins?: number;
  draws?: number;
  losses?: number;
  isMe?: boolean;
  position?: number;
  onClick?: () => void;
}

function cardGradient(rating: number) {
  if (rating >= 1200) return { from: "#7c5a00", via: "#c9960c", border: "#f0b90b" }; // gold
  if (rating >= 1100) return { from: "#1a3a6c", via: "#2255a8", border: "#4a90d9" }; // blue
  if (rating >= 1050) return { from: "#2a1a4a", via: "#4a1e8c", border: "#9b59b6" }; // purple
  return { from: "#1a2f1a", via: "#1e4a20", border: "#3a7a3a" };                     // green
}

function starsCount(rating: number) {
  if (rating >= 1200) return 5;
  if (rating >= 1100) return 4;
  if (rating >= 1050) return 3;
  if (rating >= 1020) return 2;
  return 1;
}

export function PlayerEFCard({ name, rating, rank, teamPower, wins = 0, draws = 0, losses = 0, isMe, position, onClick }: Props) {
  const g = cardGradient(rating);
  const stars = starsCount(rating);
  const initial = (name || "?").slice(0, 1).toUpperCase();
  const played = wins + draws + losses;

  return (
    <div
      onClick={onClick}
      className={cn(
        "relative w-[140px] h-[204px] rounded-2xl overflow-hidden flex-shrink-0",
        "transition-transform duration-200 hover:scale-105 hover:z-10",
        onClick && "cursor-pointer",
        isMe && "ring-2 ring-yellow-400 ring-offset-2 ring-offset-zinc-950"
      )}
      style={{ background: `linear-gradient(160deg, ${g.from}, #0c1525, ${g.from})` }}
    >
      {/* Top shimmer bar */}
      <div className="absolute top-0 left-0 right-0 h-[3px] rounded-t-2xl"
        style={{ background: `linear-gradient(90deg, transparent, ${g.border}, transparent)` }}
      />

      {/* Rating top-left */}
      <div className="absolute top-3 left-3">
        <div className="text-[28px] font-black leading-none drop-shadow-md" style={{ color: g.border }}>
          {rating}
        </div>
        <div className="text-[9px] font-bold uppercase tracking-widest text-white/50 mt-0.5">
          {rank || "ELO"}
        </div>
      </div>

      {/* Position badge top-right */}
      {position && (
        <div className="absolute top-3 right-3">
          <div className="flex h-5 w-5 items-center justify-center rounded-full text-[9px] font-black text-white"
            style={{ background: `${g.border}40`, border: `1px solid ${g.border}60` }}
          >
            {position}
          </div>
        </div>
      )}

      {/* Avatar circle */}
      <div className="absolute top-14 left-0 right-0 flex justify-center">
        <div
          className="flex items-center justify-center rounded-full text-3xl font-black text-white shadow-2xl"
          style={{
            width: 70,
            height: 70,
            background: `radial-gradient(circle at 40% 35%, ${g.border}60, ${g.from})`,
            border: `2px solid ${g.border}70`,
          }}
        >
          {initial}
        </div>
      </div>

      {/* Team power badge */}
      {teamPower ? (
        <div className="absolute top-[116px] left-0 right-0 flex justify-center">
          <div className="rounded-full px-2 py-0.5 text-[9px] font-bold text-white/70"
            style={{ background: "#00000060" }}
          >
            {teamPower.toLocaleString()}
          </div>
        </div>
      ) : null}

      {/* Bottom panel */}
      <div className="absolute bottom-0 left-0 right-0 px-2.5 pb-2.5 pt-6"
        style={{ background: `linear-gradient(to top, ${g.from}ee 60%, transparent)` }}
      >
        {/* Name */}
        <p className="text-center text-[11px] font-black text-white truncate leading-tight mb-1.5">
          {name}
        </p>

        {/* Stars */}
        <div className="flex justify-center gap-0.5 mb-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <Star key={i} size={8}
              className={i <= stars ? "fill-yellow-400 text-yellow-400" : "fill-zinc-700 text-zinc-700"}
            />
          ))}
        </div>

        {/* W/D/L stats */}
        <div className="grid grid-cols-4 gap-0.5 text-center">
          {[
            { label: "И", value: played, color: "text-white/70" },
            { label: "В", value: wins,   color: "text-green-400" },
            { label: "Н", value: draws,  color: "text-white/60" },
            { label: "П", value: losses, color: "text-red-400"   },
          ].map((s) => (
            <div key={s.label}>
              <p className="text-[7px] text-white/40 uppercase">{s.label}</p>
              <p className={cn("text-[11px] font-black leading-tight", s.color)}>{s.value}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
