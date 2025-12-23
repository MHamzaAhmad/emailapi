import { Link, useRouterState } from '@tanstack/react-router'
import { HugeiconsIcon } from '@hugeicons/react'
import {
    Home01Icon,
    Key01Icon,
    GlobeIcon,
    WebhookIcon,
    Book02Icon,
    Settings01Icon,
    PackageIcon,
} from '@hugeicons/core-free-icons'
import { useUser, UserButton } from '@clerk/clerk-react'
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarGroup,
    SidebarGroupLabel,
    SidebarGroupContent,
} from '@/components/ui/sidebar'

const mainNav = [
    { title: 'Dashboard', href: '/dashboard', icon: Home01Icon },
    { title: 'Domains', href: '/domains', icon: GlobeIcon },
    { title: 'API Keys', href: '/api-keys', icon: Key01Icon },
    { title: 'Webhooks', href: '/webhooks', icon: WebhookIcon },
]

const secondaryNav = [
    { title: 'Documentation', href: '/docs', icon: Book02Icon },
    { title: 'Settings', href: '/settings', icon: Settings01Icon },
]

export function AppSidebar() {
    const routerState = useRouterState()
    const currentPath = routerState.location.pathname
    const { user } = useUser()

    return (
        <Sidebar collapsible="icon">
            <SidebarHeader className="h-16 flex items-center px-4">
                <Link to="/" className="flex items-center gap-3">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                        <HugeiconsIcon icon={PackageIcon} size={18} />
                    </div>
                    <span className="text-sm font-bold tracking-tight group-data-[collapsible=icon]:hidden">
                        emailapi
                    </span>
                </Link>
            </SidebarHeader>

            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {mainNav.map((item) => (
                                <SidebarMenuItem key={item.href}>
                                    <SidebarMenuButton
                                        asChild
                                        isActive={currentPath === item.href}
                                        tooltip={item.title}
                                    >
                                        <Link to={item.href} className="font-medium">
                                            <HugeiconsIcon icon={item.icon} size={16} />
                                            <span>{item.title}</span>
                                        </Link>
                                    </SidebarMenuButton>
                                </SidebarMenuItem>
                            ))}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>

                <SidebarGroup className="mt-auto">
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {secondaryNav.map((item) => (
                                <SidebarMenuItem key={item.href}>
                                    <SidebarMenuButton
                                        asChild
                                        isActive={currentPath.startsWith(item.href)}
                                        tooltip={item.title}
                                    >
                                        <Link to={item.href} className="font-medium">
                                            <HugeiconsIcon icon={item.icon} size={16} />
                                            <span>{item.title}</span>
                                        </Link>
                                    </SidebarMenuButton>
                                </SidebarMenuItem>
                            ))}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>

            <SidebarFooter className="p-4 border-t border-sidebar-border">
                <div className="flex items-center gap-3 px-2">
                    <UserButton />
                    <div className="flex flex-col group-data-[collapsible=icon]:hidden">
                        <span className="text-xs font-medium truncate max-w-[120px]">
                            {user?.fullName || user?.primaryEmailAddress?.emailAddress}
                        </span>
                        <span className="text-[10px] text-muted-foreground">Premium Node</span>
                    </div>
                </div>
            </SidebarFooter>
        </Sidebar>
    )
}
