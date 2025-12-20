import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
    title: 'Email API Dashboard',
    description: 'Manage your Email API settings and analytics',
}

export default function RootLayout({
    children,
}: {
    children: React.ReactNode
}) {
    return (
        <html lang="en">
            <body>{children}</body>
        </html>
    )
}
