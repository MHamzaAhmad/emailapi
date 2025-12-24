import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
import { RootProvider } from 'fumadocs-ui/provider/tanstack'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ClerkProvider } from '@clerk/clerk-react'
import { dark } from '@clerk/themes'
import { TooltipProvider } from '@/components/ui/tooltip'
import { getQueryClient } from '@/lib/queryClient'

import appCss from '../styles.css?url'
import { ThemeProvider, useTheme } from '@/providers/themeProvider'

// Get Clerk publishable key from env
const CLERK_PUBLISHABLE_KEY = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY

if (!CLERK_PUBLISHABLE_KEY) {
  console.warn('Missing VITE_CLERK_PUBLISHABLE_KEY - auth will not work')
}

export const Route = createRootRoute({
  head: () => ({
    meta: [
      { charSet: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      { title: 'Email API' },
      { name: 'description', content: 'The simplest email API for developers' },
    ],
    links: [
      { rel: 'stylesheet', href: appCss },
      { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
    ],
  }),
  component: RootComponent,
  shellComponent: RootDocument,
})

function RootComponent() {
  return (
    <ThemeProvider attribute="class" defaultTheme="dark">
      <TooltipProvider delayDuration={300}>
        <InnerRoot />
      </TooltipProvider>
    </ThemeProvider>
  )
}

function InnerRoot() {
  const queryClient = getQueryClient()
  const { theme, systemTheme } = useTheme()
  const resolvedTheme = theme === 'system' ? systemTheme : theme

  return (
    <ClerkProvider
      publishableKey={CLERK_PUBLISHABLE_KEY || ''}
      appearance={{
        baseTheme: resolvedTheme === 'dark' ? dark : undefined,
        variables: {
          colorPrimary: 'hsl(var(--primary))',
          colorBackground: 'hsl(var(--background))',
          colorText: 'hsl(var(--foreground))',
          colorTextSecondary: 'hsl(var(--muted-foreground))',
          colorInputBackground: 'hsl(var(--secondary))',
          colorInputText: 'hsl(var(--foreground))',
          borderRadius: '0.5rem',
          fontFamily: 'Plus Jakarta Sans, sans-serif',
          fontSize: '13px',
        },
        elements: {
          card: 'bg-background border border-border shadow-2xl',
          headerTitle: 'text-foreground text-sm font-medium',
          headerSubtitle: 'text-muted-foreground text-xs',
          formButtonPrimary: 'bg-primary text-primary-foreground hover:bg-primary/90 text-xs h-8 font-medium shadow-sm transition-all',
          formFieldInput: 'bg-secondary/50 border-transparent focus:border-primary/50 transition-colors text-foreground text-xs h-8',
          formFieldLabel: 'text-foreground text-xs font-medium',
          footerActionLink: 'text-primary hover:text-primary/90 text-xs font-medium',
          dividerLine: 'bg-border/60',
          dividerText: 'text-muted-foreground/60 text-[10px] uppercase tracking-wider font-medium',
          socialButtonsBlockButton: 'bg-background text-foreground border-border hover:bg-secondary/80 transition-colors text-xs h-8 font-medium',
          socialButtonsBlockButtonText: 'text-xs font-medium',
          formFieldAction: 'text-primary hover:text-primary/90 text-xs',
          identityPreviewText: 'text-foreground text-xs font-medium',
          identityPreviewEditButton: 'text-primary text-xs',
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <Outlet />
        {import.meta.env.DEV && <ReactQueryDevtools initialIsOpen={false} />}
      </QueryClientProvider>
    </ClerkProvider>
  )
}

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <HeadContent />
      </head>
      <body>
        <RootProvider>{children}</RootProvider>
        <Scripts />
      </body>
    </html>
  )
}
