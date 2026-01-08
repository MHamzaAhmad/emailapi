import { createFileRoute, Outlet } from '@tanstack/react-router'
import { useAuth, RedirectToSignIn } from '@clerk/clerk-react'
import { Shell } from '@/components/shell'
import { SuspensionBanner } from '@/components/suspension-banner'

export const Route = createFileRoute('/_authed')({
    ssr: false, // Dashboard requires auth, skip SSR to not block landing page prerender
    component: AuthedLayout,
})

function AuthedLayout() {
    const { isLoaded, isSignedIn } = useAuth()

    if (!isLoaded) {
        // Loading state - could add a spinner here
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-pulse text-muted-foreground text-sm">Loading...</div>
            </div>
        )
    }

    if (!isSignedIn) {
        return <RedirectToSignIn />
    }

    return (
        <>
            <SuspensionBanner />
            <Shell>
                <Outlet />
            </Shell>
        </>
    )
}
