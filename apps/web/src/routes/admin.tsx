import { createFileRoute, Outlet, useNavigate } from '@tanstack/react-router'
import { useAuth, RedirectToSignIn } from '@clerk/clerk-react'
import { useCurrentUser } from '@/hooks/useUser'
import { AdminShell } from '@/components/admin-shell'
import { UserRole } from '@/generated/v1/user_pb'

export const Route = createFileRoute('/admin')({
    ssr: false, // Admin requires auth, skip SSR
    component: AdminLayout,
})

function AdminLayout() {
    const { isLoaded, isSignedIn } = useAuth()
    const { data: user, isLoading } = useCurrentUser()
    const navigate = useNavigate()

    // Loading state
    if (!isLoaded || isLoading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="animate-pulse text-muted-foreground text-sm">Loading...</div>
            </div>
        )
    }

    // Not signed in
    if (!isSignedIn) {
        return <RedirectToSignIn />
    }

    // Check admin role
    if (user && user.role !== UserRole.ADMIN) {
        // Redirect non-admins to dashboard
        navigate({ to: '/dashboard' })
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <div className="text-center space-y-2">
                    <div className="text-destructive font-semibold">Access Denied</div>
                    <div className="text-sm text-muted-foreground">You don't have admin privileges</div>
                </div>
            </div>
        )
    }

    return (
        <AdminShell>
            <Outlet />
        </AdminShell>
    )
}
