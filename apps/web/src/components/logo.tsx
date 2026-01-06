
import React from 'react'
import { cn } from '@/lib/utils'

interface LogoProps extends React.SVGProps<SVGSVGElement> {
    className?: string
}

export function Logo({ className, ...props }: LogoProps) {
    return (
        <svg
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className={cn("h-6 w-6", className)}
            {...props}
        >
            <g shapeRendering="geometricPrecision">
                {/* Unified Body Path */}
                <path
                    d="M22 2L2 10L10 13L14 22L22 2Z"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />

                {/* Central Crease */}
                <path
                    d="M10 13L22 2"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    opacity="0.5"
                />
            </g>
        </svg>
    )
}
