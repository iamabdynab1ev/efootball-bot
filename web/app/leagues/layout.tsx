import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Лиги",
  description: "Все активные лиги eFootball 2026 — регистрация, расписание и таблицы",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
