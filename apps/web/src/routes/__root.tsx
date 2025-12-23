import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
import { QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { ClerkProvider } from '@clerk/clerk-react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { getQueryClient } from '@/lib/queryClient'

import appCss from '../styles.css?url'
import { ThemeProvider } from '@/providers/themeProvider'

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
  const queryClient = getQueryClient()

  return (
    <ClerkProvider
      publishableKey={CLERK_PUBLISHABLE_KEY || ''}
      appearance={{
        baseTheme: undefined,
        variables: {
          colorPrimary: '#fff',
          colorBackground: '#000',
          colorText: '#fafafa',
          colorTextSecondary: '#888',
          colorInputBackground: '#0a0a0a',
          colorInputText: '#fafafa',
          borderRadius: '0.375rem',
          fontFamily: 'Inter, sans-serif',
          fontSize: '13px',
        },
        elements: {
          card: 'bg-background border border-border shadow-none',
          headerTitle: 'text-foreground text-sm font-medium',
          headerSubtitle: 'text-muted-foreground text-xs',
          formButtonPrimary: 'bg-primary text-primary-foreground hover:bg-primary/90 text-xs h-7',
          formFieldInput: 'bg-background border-border text-foreground text-xs h-7',
          formFieldLabel: 'text-foreground text-xs',
          footerActionLink: 'text-primary text-xs',
          dividerLine: 'bg-border',
          dividerText: 'text-muted-foreground text-2xs',
          socialButtonsBlockButton: 'bg-secondary text-secondary-foreground border-border text-xs h-8',
        },
      }}
    >
      <QueryClientProvider client={queryClient}>
        <TooltipProvider delayDuration={300}>
          <ThemeProvider attribute="class" defaultTheme="dark">
            <Outlet />
          </ThemeProvider>
        </TooltipProvider>
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
        {children}
        <Scripts />
      </body>
    </html>
  )
}
