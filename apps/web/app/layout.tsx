import type { Metadata } from "next";
import "./globals.css";
import { headers } from "next/headers";

export const metadata: Metadata = {
  title: "FileX — приватная передача файлов",
  description: "Приватное пространство для передачи файлов.",
  robots: { index: false, follow: false },
};

export default async function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  await headers();
  return <html lang="ru"><body><a className="skip-link" href="#main">К содержимому</a>{children}</body></html>;
}
