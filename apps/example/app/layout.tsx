import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: '@emailapi/sdk Example',
  description: 'Example app demonstrating the Email API SDK',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  )
}
