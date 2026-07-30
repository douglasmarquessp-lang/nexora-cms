import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: {
    default: "Nexora CMS — Plataforma de Conteúdo com IA",
    template: "%s | Nexora CMS",
  },
  description: "Plataforma de gerenciamento de conteúdo com inteligência artificial. Publique, gerencie e otimize seu conteúdo editorial.",
  openGraph: {
    title: "Nexora CMS",
    description: "Plataforma de gerenciamento de conteúdo com inteligência artificial.",
    type: "website",
    locale: "pt_BR",
    siteName: "Nexora CMS",
  },
  twitter: {
    card: "summary_large_image",
    title: "Nexora CMS",
    description: "Plataforma de gerenciamento de conteúdo com inteligência artificial.",
  },
  robots: {
    index: true,
    follow: true,
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="pt-BR">
      <body className="min-h-screen bg-white text-gray-900 antialiased">
        {children}
      </body>
    </html>
  );
}
