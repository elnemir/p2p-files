import type { ReactNode } from "react";
import "./style.css";
export const metadata = {
  title: "FileX — панель издателя лицензий",
  description: "Закрытый контур выдачи офлайн-лицензий FileX",
};
export default function Layout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru">
      <body>{children}</body>
    </html>
  );
}
