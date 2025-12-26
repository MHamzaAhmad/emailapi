import { HeadContent, Outlet, Scripts, createRootRoute } from '@tanstack/react-router'
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
        <script
          dangerouslySetInnerHTML={{
            __html: `
              !function(t,e){var o,n,p,r;e.__SV||(window.posthog && window.posthog.__loaded)||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.crossOrigin="anonymous",p.async=!0,p.src=s.api_host.replace(".i.posthog.com","-assets.i.posthog.com")+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="init Xr es pi Zr rs Kr Qr capture Ni calculateEventProperties os register register_once register_for_session unregister unregister_for_session ds getFeatureFlag getFeatureFlagPayload isFeatureEnabled reloadFeatureFlags updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures on onFeatureFlags onSurveysLoaded onSessionId getSurveys getActiveMatchingSurveys renderSurvey displaySurvey cancelPendingSurvey canRenderSurvey canRenderSurveyAsync identify setPersonProperties group resetGroups setPersonPropertiesForFlags resetPersonPropertiesForFlags setGroupPropertiesForFlags resetGroupPropertiesForFlags reset get_distinct_id getGroups get_session_id get_session_replay_url alias set_config startSessionRecording stopSessionRecording sessionRecordingStarted captureException startExceptionAutocapture stopExceptionAutocapture loadToolbar get_property getSessionProperty us ns createPersonProfile hs Vr vs opt_in_capturing opt_out_capturing has_opted_in_capturing has_opted_out_capturing get_explicit_consent_status is_capturing clear_opt_in_out_capturing ss debug O ls getPageViewId captureTraceFeedback captureTraceMetric qr".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);

              posthog.init('phc_2626jWn9VFnpMMlY9JeccLaH9kV1BEMvtpKTq5Cbmz7', {
                  api_host: 'https://us.i.posthog.com',
                  defaults: '2025-11-30',
                  person_profiles: 'identified_only',
              })
            `,
          }}
        />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  )
}
