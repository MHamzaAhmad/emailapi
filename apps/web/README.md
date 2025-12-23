# EmailAPI Dashboard

The frontend dashboard for the EmailAPI platform, built with modern web technologies to monitor email activity, domains, and webhooks.

## Getting Started

To run this application:

```bash
pnpm install
pnpm start
```

## Building For Production

To build this application for production:

```bash
pnpm build
```

## Tech Stack

- **Framework**: [TanStack Start / Router](https://tanstack.com/router)
- **Styling**: [Tailwind CSS](https://tailwindcss.com/)
- **UI Components**: [Shadcn UI](https://ui.shadcn.com/)
- **Data Fetching**: [TanStack Query](https://tanstack.com/query)
- **State Management**: [TanStack Store](https://tanstack.com/store)

## Design System

This project enforces a specific design language for consistency across components, particularly inputs and buttons.

### Sizing (Compact)

Components use a compact sizing scale to maintain high density:

- **Default**: `h-7` (28px). Applies to standard Buttons and Select inputs.
- **Small (sm)**: `h-6` (24px). Used for secondary actions or dense toolbars.
- **Large (lg)**: `h-8` (32px). Used for primary filter triggers (e.g., Dashboard Filters).

### Variants

- **Dashed**: A custom variant (`variant="dashed"`) or class (`border-dashed`) is used for filter triggers and optional inputs to distinguish them from primary form fields.
- **Outline**: The default style for secondary actions.

### Consistency Rules

1.  **Select & Button Alignment**: `Select` components have been customized to match `Button` sizes exactly. Always use the corresponding size prop (`size="sm"`, `size="lg"`) to align them in a row.
2.  **Icons**: Use [HugeIcons](https://hugeicons.com/) (via `@hugeicons/react`) for all new icons.
