'use client'

import { useEffect, useRef } from 'react'

// PostHog configuration
const POSTHOG_KEY = 'phc_2626jWn9VFnpMMlY9JeccLaH9kV1BEMvtpKTq5Cbmz7'
const POSTHOG_HOST = 'https://us.i.posthog.com'

/**
 * Deferred PostHog loader - initializes PostHog after page becomes interactive.
 * This improves LCP and TBT by not blocking initial render.
 */
export function DeferredPostHog() {
    const initialized = useRef(false)

    useEffect(() => {
        if (initialized.current) return
        initialized.current = true

        // Defer loading until browser is idle or after 2 seconds (whichever comes first)
        const loadPostHog = () => {
            // Create the PostHog stub if it doesn't exist
            if (typeof window !== 'undefined' && !window.posthog) {
                const posthog: any = []
                posthog._i = []

                posthog.init = function (apiKey: string, config: any, name?: string) {
                    const u = name ? posthog[name] = [] : posthog
                    u.people = u.people || []
                    u.toString = function (includeStub?: boolean) {
                        let str = 'posthog'
                        if (name && name !== 'posthog') str += '.' + name
                        if (!includeStub) str += ' (stub)'
                        return str
                    }
                    u.people.toString = function () { return u.toString(true) + '.people (stub)' }

                    // Stub all PostHog methods
                    const methods = [
                        'init', 'capture', 'identify', 'register', 'register_once', 'unregister',
                        'getFeatureFlag', 'isFeatureEnabled', 'reloadFeatureFlags', 'on', 'onFeatureFlags',
                        'onSurveysLoaded', 'getSurveys', 'getActiveMatchingSurveys', 'renderSurvey',
                        'opt_in_capturing', 'opt_out_capturing', 'has_opted_in_capturing', 'has_opted_out_capturing',
                        'reset', 'get_distinct_id', 'getGroups', 'get_session_id', 'alias', 'set_config',
                        'startSessionRecording', 'stopSessionRecording', 'sessionRecordingStarted',
                        'debug', 'getPageViewId', 'group', 'setPersonProperties', 'resetGroups'
                    ]

                    methods.forEach(method => {
                        u[method] = function (...args: any[]) {
                            u.push([method, ...args])
                        }
                    })

                    posthog._i.push([apiKey, config, name])
                }

                posthog.__SV = 1
                window.posthog = posthog
            }

            // Initialize PostHog
            window.posthog?.init(POSTHOG_KEY, {
                api_host: POSTHOG_HOST,
                defaults: '2025-11-30',
                person_profiles: 'identified_only',
                loaded: () => {
                    // Load the actual PostHog script after initialization
                    const script = document.createElement('script')
                    script.type = 'text/javascript'
                    script.crossOrigin = 'anonymous'
                    script.async = true
                    script.src = POSTHOG_HOST.replace('.i.posthog.com', '-assets.i.posthog.com') + '/static/array.js'
                    document.head.appendChild(script)
                }
            })
        }

        // Use requestIdleCallback if available, otherwise setTimeout
        if ('requestIdleCallback' in window) {
            requestIdleCallback(loadPostHog, { timeout: 2000 })
        } else {
            setTimeout(loadPostHog, 1500)
        }
    }, [])

    return null
}

// Type augmentation for window.posthog
declare global {
    interface Window {
        posthog?: any
    }
}
