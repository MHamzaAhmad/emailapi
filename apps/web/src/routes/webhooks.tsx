import { createFileRoute } from '@tanstack/react-router'
import { AppPortal } from 'svix-react'
import 'svix-react/style.css'
import { Shell } from '@/components/shell'
import { AuthGuard } from '@/components/auth-guard'
import { useWebhookPortal } from '@/hooks'
import { useEffect, useState } from 'react'

export const Route = createFileRoute('/webhooks')({
    component: WebhooksPage,
})

function WebhooksPage() {
    return (
        <AuthGuard>
            <WebhooksContent />
        </AuthGuard>
    )
}

function WebhooksContent() {
    const { data, isLoading, isError, error } = useWebhookPortal()
    const [isDarkMode, setIsDarkMode] = useState(false)

    // Detect dark mode
    useEffect(() => {
        const checkDarkMode = () => {
            setIsDarkMode(document.documentElement.classList.contains('dark'))
        }
        checkDarkMode()

        // Watch for dark mode changes
        const observer = new MutationObserver(checkDarkMode)
        observer.observe(document.documentElement, {
            attributes: true,
            attributeFilter: ['class'],
        })

        return () => observer.disconnect()
    }, [])

    // Build portal URL with theming parameters
    const getThemedUrl = () => {
        if (!data?.url) return undefined

        const url = new URL(data.url)

        // Apply custom theming via URL parameters
        // Primary color: Premium Indigo (matches --primary: 226 70% 60%)
        if (isDarkMode) {
            url.searchParams.set('primaryColorDark', '6882d1')
            url.searchParams.set('darkMode', 'true')
        } else {
            url.searchParams.set('primaryColorLight', '5472d4')
        }

        return url.toString()
    }

    return (
        <Shell>
            <div className="relative min-h-[600px] mt-4">
                {isLoading && (
                    <div className="absolute inset-0 flex items-center justify-center">
                        <div className="space-y-4 w-full max-w-md text-center">
                            <div className="animate-pulse flex flex-col gap-3 items-center">
                                <div className="h-3 bg-muted rounded w-48"></div>
                                <div className="h-2 bg-muted/50 rounded w-32"></div>
                            </div>
                            <p className="text-xs text-muted-foreground">Loading webhook portal...</p>
                        </div>
                    </div>
                )}

                {isError && (
                    <div className="border border-destructive/20 bg-destructive/5 p-8 text-center rounded-lg">
                        <p className="text-sm text-destructive font-medium mb-2">
                            Failed to load webhook portal
                        </p>
                        <p className="text-xs text-muted-foreground">
                            {error?.message || 'Please try again later.'}
                        </p>
                    </div>
                )}

                {data?.url && !isLoading && (
                    <div className="rounded-xl border border-border/40 bg-card overflow-hidden">
                        <AppPortal
                            url={getThemedUrl()}
                            fullSize
                        />
                    </div>
                )}
            </div>
        </Shell>
    )
}
