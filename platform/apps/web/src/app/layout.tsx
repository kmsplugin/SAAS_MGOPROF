import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import { QueryProvider } from '@/components/layout/query-provider'
import './globals.css'

const inter = Inter({ subsets: ['latin', 'cyrillic'] })

export const metadata: Metadata = {
  title: {
    default: 'Platform — Online Events & Webinars',
    template: '%s | Platform',
  },
  description: 'Professional online events, webinars and broadcasts platform.',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru" suppressHydrationWarning>
      <body className={inter.className}>
        <QueryProvider>{children}</QueryProvider>
      </body>
    </html>
  )
}
