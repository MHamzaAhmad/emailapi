/**
 * HugeIcons wrapper component
 * Uses @hugeicons/react with @hugeicons/core-free-icons
 */
import { HugeiconsIcon } from '@hugeicons/react'
import type { IconSvgElement } from '@hugeicons/react'

// Re-export commonly used icons
export {
    Home01Icon,
    Key01Icon,
    GlobalIcon,
    WebhookIcon,
    Book02Icon,
    Settings01Icon,
    ArrowRight01Icon,
    Add01Icon,
    MoreHorizontalIcon,
    Copy01Icon,
    Delete01Icon,
    Mail01Icon,
    CheckmarkCircle01Icon,
    AlertCircleIcon,
    Clock01Icon,
    Loading01Icon,
    RefreshIcon,
    Tap01Icon,
    CodeIcon,
    Shield01Icon,
} from '@hugeicons/core-free-icons'

interface IconProps {
    icon: IconSvgElement
    size?: number
    className?: string
    strokeWidth?: number
}

export function Icon({ icon, size = 14, className, strokeWidth = 1.5 }: IconProps) {
    return (
        <HugeiconsIcon
            icon={icon}
            size={size}
            color="currentColor"
            strokeWidth={strokeWidth}
            className={className}
        />
    )
}
