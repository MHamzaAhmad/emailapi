import { createFileRoute, Outlet } from '@tanstack/react-router'
import { PublicNavbar } from '@/components/public-navbar'
import { PublicFooter } from '@/components/public-footer'

export const Route = createFileRoute('/_public')({
    component: PublicLayout,
})

function PublicLayout() {
    return (
        <div className="min-h-screen bg-background text-foreground font-sans flex flex-col antialiased overflow-x-hidden w-full">
            <PublicNavbar />
            <div className="flex-1">
                <Outlet />
            </div>
            <PublicFooter />
        </div>
    )
}
