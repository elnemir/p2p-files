import type { Metadata } from "next";
import "./globals.css";
import { headers } from "next/headers";
import { readBranding } from "../lib/branding-server";
import { DEFAULT_FAVICON } from "../lib/branding";
import BrandingProvider from "../components/BrandingProvider";
export async function generateMetadata(): Promise<Metadata> {
  const { value: b } = await readBranding();
  return {
    title: `${b.productName} — ${b.taglineRu}`,
    description: b.taglineRu,
    robots: { index: false, follow: false },
    icons: {
      icon: b.favicon || DEFAULT_FAVICON,
      apple: b.favicon || DEFAULT_FAVICON,
    },
  };
}
export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  await headers();
  const initial = await readBranding();
  return (
    <html lang="ru" data-theme={initial.value.defaultTheme}>
      <body>
        <BrandingProvider initial={initial}>
          <a className="skip-link" href="#main">
            К содержимому
          </a>
          {children}
        </BrandingProvider>
      </body>
    </html>
  );
}
