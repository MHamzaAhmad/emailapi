"use client"

import { HugeiconsIcon } from "@hugeicons/react"
import { Moon02Icon, Sun03Icon } from "@hugeicons/core-free-icons"
import { useTheme } from "@/providers/themeProvider"

import { Button } from "@/components/ui/button"

export function ModeToggle() {
    const { theme, setTheme } = useTheme()

    // Toggle between light and dark
    const toggleTheme = () => {
        setTheme(theme === "dark" ? "light" : "dark")
    }

    return (
        <Button
            variant="outline"
            size="sm"
            onClick={toggleTheme}
            className="h-8 w-8 px-0 border-dashed hover:border-solid hover:bg-secondary/50 transition-all duration-200"
        >
            <div className="relative h-[1.2rem] w-[1.2rem]">
                <div className="absolute inset-0 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0">
                    <HugeiconsIcon icon={Sun03Icon} size={18} className="text-muted-foreground" />
                </div>
                <div className="absolute inset-0 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100">
                    <HugeiconsIcon icon={Moon02Icon} size={18} className="text-foreground" />
                </div>
            </div>
            <span className="sr-only">Toggle theme</span>
        </Button>
    )
}
