import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Профиль",
  description: "Мой профиль, статистика и история матчей eFootball",
};

export default function Layout({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
