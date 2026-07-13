# StockRush Go Visual System

## Scene

A technical reviewer scans the project on a laptop in a bright office, with ten minutes to verify a bold concurrency claim; the interface must read like crisp evidence on paper.

## Theme

Light, restrained, and warm. Tinted paper neutrals carry the surface; rust marks actions; dark green is reserved for verified invariants.

## Colors

- Canvas: `oklch(0.975 0.008 82)`
- Surface: `oklch(0.995 0.004 82)`
- Ink: `oklch(0.235 0.018 55)`
- Muted ink: `oklch(0.49 0.018 55)`
- Rule: `oklch(0.86 0.018 75)`
- Action: `oklch(0.55 0.16 35)`
- Action hover: `oklch(0.49 0.16 35)`
- Pass: `oklch(0.46 0.11 145)`
- Warning: `oklch(0.63 0.13 80)`
- Error: `oklch(0.52 0.18 25)`

## Typography

Use the native system sans stack. Use tabular numerals for inventory, latency, and counts. Body copy stays below 72 characters per line where prose is present.

## Layout

A compact top navigation leads into an evidence ledger. Sections use rules and whitespace instead of repeated cards. Tables scroll horizontally on narrow screens; the inventory equation collapses vertically below 720px.

## Components

- Buttons: one consistent shape, 44px minimum target, solid rust for primary and outlined neutral for secondary.
- Status: text plus semantic color; never color alone.
- Data rows: aligned tabular numbers and explicit labels.
- Loading: short skeleton rows.
- Focus: visible two-pixel action-colored outline with offset.

## Motion

Only state transitions use motion, 180ms ease-out. Respect `prefers-reduced-motion` and avoid page-load choreography.
