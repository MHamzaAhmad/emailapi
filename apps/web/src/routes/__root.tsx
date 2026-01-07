import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ClerkProvider } from '@clerk/clerk-react'
import { dark } from '@clerk/themes'
import { TooltipProvider } from '@/components/ui/tooltip'
import { getQueryClient } from '@/lib/queryClient'

import appCss from '../styles.css?url'
import { ThemeProvider, useTheme } from '@/providers/themeProvider'
import { NotFound } from '@/components/NotFound'
import { ErrorComponent } from '@/components/ErrorComponent'

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
      { title: 'SimpleEmailAPI - The simplest API to send and receive emails' },
      { name: 'description', content: 'The simplest API to send and receive emails.' },

      // Open Graph
      { property: 'og:title', content: 'SimpleEmailAPI' },
      { property: 'og:description', content: 'The simplest API to send and receive emails.' },
      { property: 'og:image', content: '/ogimage.png' },
      { property: 'og:type', content: 'website' },

      // Twitter
      { name: 'twitter:card', content: 'summary_large_image' },
      { name: 'twitter:title', content: 'SimpleEmailAPI' },
      { name: 'twitter:description', content: 'The simplest API to send and receive emails.' },
      { name: 'twitter:image', content: '/ogimage.png' },
    ],
    links: [
      // Preconnect to critical origins (~300ms LCP savings)
      { rel: 'preconnect', href: 'https://clerk.simpleemailapi.dev' },
      { rel: 'preconnect', href: 'https://us.i.posthog.com' },
      { rel: 'preconnect', href: 'https://us-assets.i.posthog.com', crossOrigin: 'anonymous' },
      // Preload critical CSS to reduce render blocking
      { rel: 'preload', as: 'style', href: appCss },
      { rel: 'stylesheet', href: appCss },
      { rel: 'icon', type: 'image/png', href: '/favicon-96x96.png', sizes: '96x96' },
      { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
      { rel: 'shortcut icon', href: '/favicon.ico' },
      { rel: 'apple-touch-icon', sizes: '180x180', href: '/apple-touch-icon.png' },
      { rel: 'manifest', href: '/site.webmanifest' },
    ],
  }),
  component: RootComponent,
  shellComponent: RootDocument,
  notFoundComponent: NotFound,
  errorComponent: ErrorComponent,
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

import { DeferredPostHog } from '@/components/deferred-posthog'

function RootDocument({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <HeadContent />
      </head>
      <body>
        {children}
        <DeferredPostHog />
        <Scripts />
      </body>
    </html>
  )
}

