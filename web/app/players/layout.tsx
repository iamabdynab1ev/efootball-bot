import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Игроки",
  description: "Рейтинг игроков eFootball: ELO, Win Rate, серии побед, бомбардиры",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
