---
name: Nocu
description: A quiet control room for NOC operators, where one orange signal means something needs you.
colors:
  signal-orange: "#f56e2c"
  signal-orange-deep: "#ca5c24"
  focus-ring: "#ca5c24"
  on-signal: "oklch(0.145 0 0)"
  warm-cream: "#fdf0dd"
  on-warm-cream: "#5a2e12"
  clear-sky-blue: "#87aecd"
  on-clear-sky-blue: "#0f2a40"
  alarm-red: "oklch(0.577 0.245 27.325)"
  paper: "oklch(1 0 0)"
  ink: "oklch(0.145 0 0)"
  ink-soft: "oklch(0.556 0 0)"
  surface-muted: "oklch(0.97 0 0)"
  surface-sidebar: "oklch(0.985 0 0)"
  hairline: "oklch(0.922 0 0)"
typography:
  headline:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: "2rem"
    letterSpacing: "-0.025em"
  title:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: "1.5rem"
    letterSpacing: "0em"
  section-label:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: "1.25rem"
    letterSpacing: "0em"
  body:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: "1.25rem"
    letterSpacing: "0em"
  label:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    lineHeight: "1rem"
    letterSpacing: "0em"
  nav:
    fontFamily: "Manrope, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 500
    lineHeight: "1.5rem"
    letterSpacing: "0em"
  data:
    fontFamily: "Geist Mono, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: "1rem"
    letterSpacing: "0em"
rounded:
  sm: "6px"
  md: "8px"
  lg: "10px"
  xl: "14px"
  pill: "26px"
  full: "9999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "24px"
  2xl: "32px"
  3xl: "40px"
components:
  button-primary:
    backgroundColor: "{colors.signal-orange}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.lg}"
    padding: "0 10px"
    height: "32px"
  button-primary-hover:
    backgroundColor: "{colors.signal-orange-deep}"
    textColor: "{colors.ink}"
  button-outline:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.lg}"
    padding: "0 10px"
    height: "32px"
  button-outline-hover:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.ink}"
  button-secondary:
    backgroundColor: "{colors.warm-cream}"
    textColor: "{colors.on-warm-cream}"
    typography: "{typography.body}"
    rounded: "{rounded.lg}"
    padding: "0 10px"
    height: "32px"
  button-ghost:
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.lg}"
    padding: "0 10px"
    height: "32px"
  button-ghost-hover:
    backgroundColor: "{colors.surface-muted}"
    textColor: "{colors.ink}"
  button-sm:
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "0 10px"
    height: "28px"
  badge-signal:
    backgroundColor: "{colors.signal-orange}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "2px 8px"
    height: "20px"
  badge-healthy:
    backgroundColor: "{colors.clear-sky-blue}"
    textColor: "{colors.on-clear-sky-blue}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "2px 8px"
    height: "20px"
  badge-cream:
    backgroundColor: "{colors.warm-cream}"
    textColor: "{colors.on-warm-cream}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "2px 8px"
    height: "20px"
  badge-outline:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "2px 8px"
    height: "20px"
  card:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.xl}"
    padding: "16px"
  input:
    backgroundColor: "{colors.paper}"
    textColor: "{colors.ink}"
    typography: "{typography.body}"
    rounded: "{rounded.lg}"
    padding: "4px 10px"
    height: "32px"
  tooltip:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "6px 12px"
  nav-item:
    textColor: "{colors.ink-soft}"
    typography: "{typography.nav}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "44px"
  nav-item-active:
    textColor: "{colors.ink}"
    typography: "{typography.nav}"
    rounded: "{rounded.md}"
    padding: "0 12px"
    height: "44px"
---

# Design System: Nocu

## Overview

**Creative North Star: "The Quiet Control Room"**

Nocu is designed like a dimmed operations floor. At rest the whole interface is neutral: white paper, a barely-lighter sidebar, hairline dividers, and grey text at two weights. Nothing competes for the eye, because the eye is being saved for one thing. When a monitor goes down or an incident opens, a single hot colour appears, Signal Orange, and because it is the only saturated warm colour on screen, it cannot be missed. Healthy states are marked in a cool, low-chroma Clear Sky Blue so that "fine" reads as calm rather than celebratory. Everything else is grey. Attention is engineered by rarity.

The system is precise and warm at the same time. Precision comes from the geometry: thirty-two-pixel controls, ten-pixel corners, two-pixel severity bars, six-pixel status dots, monospace timestamps aligned in a fixed column, and surfaces bounded by a one-pixel hairline rather than a shadow. Warmth comes from the material: Manrope's rounded humanist letterforms, a Warm Cream tint for secondary surfaces and avatars, and an orange whose lightness keeps it from reading as danger. Red exists but is held back for the two places it is literally true: critical severity and destructive actions.

Density is moderate. Operators keep this screen open for a whole shift, so rows are tall enough to scan at a glance across a desk, but page chrome is minimal and in-page surfaces are flat. Motion is short and functional: overlays fade and scale in over a tenth of a second, buttons press down one pixel, and the only ambient animation is the live-connection dot's pulse. When the viewer prefers reduced motion the pulse stops and overlays keep only their fade: no slide, zoom, or icon rotation.

**Key Characteristics:**
- One saturated accent, Signal Orange, meaning "needs you"; its rarity is the whole point.
- Cool blue for healthy, neutral grey for nothing to see, red only for critical and destructive.
- Flat, hairline-bounded surfaces; shadows only on things floating above the page.
- Thin marks (dots, bars, strips) instead of filled panels to carry status.
- Manrope for everything human; Geist Mono for everything machine (URLs, times, latency).
- Compact and exact controls: 32px tall, 10px corners, pill badges.

## Colors

A grey room with one orange lamp and one blue lamp, on paper that turns to slate in dark mode.

### Primary
- **Signal Orange** (`#f56e2c`): the only hot colour in the system. Primary action buttons, the active navigation marker, the "down" monitor badge, the "high" severity bar, and the "reconnecting" dot. It means a human is wanted. Deepens to **Signal Orange Deep** (`#ca5c24`) on hover; in dark mode the hover is slightly lighter (`#e0692b`) so it still reads as a change. Text on any orange fill is Ink (see The Ink On Orange Rule). Signal Orange Deep is also the light-mode focus ring (`focus-ring`, 4.1:1 on Paper); dark mode rings use Signal Orange (6.1:1 on slate).

### Secondary
- **Warm Cream** (`#fdf0dd`) with **Cocoa** text (`#5a2e12`): a warm tinted surface for things that should sit slightly forward without alarm. The "open" incident badge, the "pending" monitor badge, role chips, the avatar initial circle, and sidebar hover. In dark mode the pair inverts to a cocoa surface (`#3a2a1e`) with cream text.

### Tertiary
- **Clear Sky Blue** (`#87aecd`) with **Deep Navy** text (`#0f2a40`): healthy. The "up" monitor badge, up heartbeats, the live-connection dot, the "medium" severity bar. Deliberately low in chroma so good news stays quiet. Dark mode uses a slightly deeper blue (`#6f9cc0`).

### Neutral
- **Paper** (`oklch(1 0 0)`): page and card background. Dark mode: `oklch(0.145 0 0)` page, `oklch(0.205 0 0)` cards and popovers.
- **Ink** (`oklch(0.145 0 0)`): primary text, headings, and the inverted tooltip surface. Dark mode: `oklch(0.985 0 0)`.
- **Ink Soft** (`oklch(0.556 0 0)`): secondary text, section labels, inactive navigation, timestamps. Dark mode: `oklch(0.708 0 0)`.
- **Surface Muted** (`oklch(0.97 0 0)`): row hover, tab list track, empty heartbeat slots, ghost-button hover. Dark mode: `oklch(0.269 0 0)`.
- **Surface Sidebar** (`oklch(0.985 0 0)`): the sidebar and mobile header, one step off Paper so the chrome is distinguishable without a hard edge. Dark mode: `oklch(0.205 0 0)`.
- **Hairline** (`oklch(0.922 0 0)`): all borders, dividers, and input strokes. Dark mode: `oklch(1 0 0 / 10%)` borders, `oklch(1 0 0 / 15%)` input strokes.
- **Alarm Red** (`oklch(0.577 0.245 27.325)`): the critical severity bar in incident lists, the "down" backend-health badge, destructive buttons (as a 10% tint with red text, never a solid fill), and invalid field rings. Dark mode: `oklch(0.704 0.191 22.216)`.

### Named Rules
**The One Lamp Rule.** Signal Orange appears on a screen only where an operator is wanted: the primary action, the active nav marker, and down/high/critical status. It is never used for decoration, headings, icons at rest, or hover tints on neutral elements. If a screen has nothing wrong, the only orange on it is the active nav marker and at most one primary button.

**The Cool Means Fine Rule.** Healthy states use Clear Sky Blue, never green. Green is not in the palette and must not be introduced for "up" or "success"; success toasts stay neutral.

**The Red Is Literal Rule.** Alarm Red is reserved for critical severity and destructive or invalid states. Warnings and "down" use Signal Orange, not red.

**The Ink On Orange Rule.** Text and icons set on a Signal Orange fill are Ink, never white. White on this orange measures 2.9:1; Ink measures 6.7:1 and reads like a stencilled label on a signal lamp.

**The Solid Ring Rule.** The focus ring is a solid 2px ring of Signal Orange Deep in light mode and Signal Orange in dark mode, never a translucent halo. Both reach at least 3:1 on Paper and on the sidebar.

## Typography

**Display Font:** Manrope (variable weight, with system-ui, sans-serif)
**Body Font:** Manrope (same family; headings differ only in size and weight)
**Label/Mono Font:** Geist Mono (with ui-monospace, Menlo, monospace)

**Character:** A single humanist sans at two weights does almost all the work, so hierarchy comes from size, weight, and grey level rather than from a second display face. Manrope's wide, rounded letterforms carry the warmth; Geist Mono carries the machine data (URLs, timestamps, latency) and signals "this is a value, not a sentence" without any extra styling.

### Hierarchy
- **Headline** (600, 1.5rem, 2rem, tracking -0.025em): one per page. The page title, the monitor name, the incident title, and on the overview the live situation sentence, where the count inside it is coloured Signal Orange. Long titles wrap with balanced lines.
- **Title** (600, 1rem, 1.5rem): section headings inside a page ("Check history", "Last 30 checks", form section titles) and card titles (500 weight in cards).
- **Section Label** (500, 0.875rem, 1.25rem, Ink Soft): the quiet heading over a list on the overview ("Needs attention", "Live events"). Grey and the same size as body so it labels without competing with the rows under it. Never uppercase, never letter-spaced.
- **Body** (400, 0.875rem, 1.25rem): everything else, including table cells, list rows, descriptions, and button labels (500 weight). Inputs render body size on desktop and 1rem on touch widths so iOS does not zoom. Prose runs at most a `max-w-prose` measure.
- **Label** (500, 0.75rem, 1rem): badges, secondary lines under a row title, timestamps, field hints. Sentence case.
- **Nav** (500, 1rem, 1.5rem): sidebar navigation labels only. One step above body so the three destinations stay readable from across a desk without pulling weight from the page heading.
- **Data** (Geist Mono, 400, 0.75rem to 0.875rem): URLs, relative and absolute times, latency, request details. Always Ink Soft unless it is the primary content of the row.

### Named Rules
**The Two Weights Rule.** Manrope is used at 400 and 500 for text and 600 for headings. Nothing is bold (700) and nothing is light (300).

**The Mono Is a Value Rule.** Geist Mono is applied to machine-produced strings only. It is never used for headings, labels, or UI copy to look technical.

## Layout

A fixed sidebar with a centred content column. On wide screens (768px and up) a 256px (`w-64`) sticky sidebar on Surface Sidebar holds the brand, three navigation items, and a footer with the account menu and theme toggle, separated from content by a hairline. Content sits in a column padded 24px (40px from 1024px) horizontally and 32px vertically, capped at 1152px (`max-w-6xl`) and centred. Below 768px the sidebar becomes a 56px top bar with a hamburger that opens the same navigation in a left sheet.

Pages are vertical stacks. The page header (headline plus an optional subline or badges, and a right-aligned action row) is followed by sections separated by 24px to 40px (`space-y-6`, `-8`, `-10`); the overview uses the largest gap and a two-column grid from 1024px for "Needs attention" and "Live events". Detail pages use a key-value grid that goes from two to four columns (`grid-cols-2 sm:grid-cols-3 lg:grid-cols-4`, 24px column gap, 16px row gap) and a two-column body from 768px. Forms are capped at 768px (`max-w-3xl`), grouped into sections divided by a hairline top border with 32px padding above, and lay out fields in two- or three-column grids with 20px gaps.

Rows in lists and tables are 12px padded top and bottom (`py-3`) with hairline dividers (`divide-y`) and no outer border; the row's hover is a 50% Surface Muted wash. Table headers are 40px tall. Controls use a 6px gap between icon and label and 8px between sibling controls. The spacing scale is 4px based; the steps that recur are 4, 8, 12, 16, 24, 32, and 40.

## Elevation & Depth

Flat by default, lifted only for overlays. In-page surfaces (cards, panels, tables, forms) never cast a shadow: a card is bounded by a one-pixel ring at 10% Ink (`ring-1 ring-foreground/10`), a panel by a Hairline border, and lists by hairline dividers alone. Depth within the page is tonal: the sidebar is one step off Paper, hovered rows and the tab track are Surface Muted, and card footers sit on a 50% Surface Muted wash. Dark mode keeps the same relationships with the hairline becoming a 10% white line.

Shadows appear only on elements that float above the page and belong to the overlay layer.

### Shadow Vocabulary
- **Popover** (`shadow-md`, Tailwind default): dropdown menus, selects, popovers, and the command-style menus that open from a trigger.
- **Dialog** (`shadow-lg`, Tailwind default): modal dialogs, alert dialogs, and the mobile navigation sheet, paired with a 50% black scrim.
- **Active tab** (`shadow-sm`): the selected trigger in a default tab list, the one in-page exception, because the trigger is conceptually lifted off the muted track.

### Named Rules
**The Flat Floor Rule.** Nothing that scrolls with the page casts a shadow. If a new in-page surface needs separation, use the 10% ring or a Hairline border, or a tonal shift to Surface Muted.

## Shapes

One base radius of 10px (`--radius: 0.625rem`) drives a geometric scale: 6px small, 8px medium, 10px large, 14px extra-large, and a 26px pill. Buttons, inputs, selects, and panels use the 10px corner; small and extra-small buttons step down to 8px and 6px so the corner stays proportional to the height; cards use 14px; badges are full pills; avatars, status dots, and severity bars are fully round. Overlays (dropdowns, dialogs, tooltips) use 8px to 10px. There are no sharp corners and no corner larger than the pill anywhere in the system.

Borders are always one pixel and always Hairline. Status is carried by thin geometry rather than filled shapes: a 2px-wide vertical bar for severity, a 6px dot for monitor status and event tone, a 3px rounded marker on the left edge of the active nav item, and 4px-wide bars for heartbeats. Nothing is outlined in colour; colour fills the mark itself.

## Components

Compact and exact. Every control is 32px tall at its default size, labels are 0.875rem at 500 weight, icons are 16px, and every interactive element shares the same focus treatment: the border turns to the ring colour and a solid 2px ring appears (Signal Orange Deep in light mode, Signal Orange in dark). Disabled controls drop to 50% opacity and lose pointer events. Nothing scales up on hover; the only press feedback is a one-pixel downward translate on buttons.

### Buttons
- **Shape:** gently rounded (10px); small 28px buttons use 8px corners and 0.8rem text; extra-small 24px buttons use 6px corners and 0.75rem text; icon buttons are square at the same heights.
- **Primary:** Signal Orange fill with Ink text, 10px horizontal padding, 6px gap to an icon. Hover deepens to Signal Orange Deep. At most one per view.
- **Outline:** Paper fill with a Hairline border and Ink text; hover and open states wash to Surface Muted. The default for secondary actions ("Refresh", "Try again", "Back to monitors").
- **Secondary:** Warm Cream fill with Cocoa text; hover mixes 5% Ink into the cream. Used sparingly for warm, non-urgent affordances.
- **Ghost:** no fill, Ink text; hover washes to Surface Muted. Used for icon buttons in chrome (theme toggle, hamburger, account menu, row overflow menus).
- **Destructive:** Alarm Red at 10% tint with Alarm Red text, never a solid red fill; hover 20%. Focus ring is red.
- **Link:** Signal Orange text, underline on hover with a 4px offset.
- **Role gating:** a control the viewer may not use is rendered disabled and wrapped in a focusable span that shows the tooltip "Needs the operator role" on hover and focus.

### Badges
- **Style:** 20px pill, 0.75rem at 500 weight, 8px horizontal padding, optional 12px icon.
- **Signal** (orange fill, Ink text): monitor "Down".
- **Healthy** (Clear Sky Blue fill, Deep Navy text): monitor "Up".
- **Cream** (Warm Cream fill, Cocoa text): monitor "Pending", incident "Open", role chips.
- **Outline** (Hairline border, Ink text): neutral states such as "Paused", "Acknowledged", "Resolved", "Closed".
- **Destructive** (10% Alarm Red tint, red text): backend service "down".

### Cards / Containers
- **Corner Style:** 14px.
- **Background:** Paper (slate in dark mode), 0.875rem text.
- **Shadow Strategy:** none; a 1px ring at 10% Ink (see Elevation).
- **Border:** the ring is the border. Panels that are not cards (error, not-found, form sub-groups) use a Hairline border with a 10px corner and 16px padding instead.
- **Internal Padding:** 16px (`--card-spacing`), 12px for the small size, applied to header, content, and footer; the footer sits on a 50% Surface Muted wash behind a top hairline.
- **Use:** cards are for self-contained units (the sign-in form, backend health). Lists and tables on a page sit directly on Paper with dividers, not inside cards.

### Inputs / Fields
- **Style:** 32px tall, 10px corners, transparent fill with a Hairline stroke, 10px horizontal padding, Ink Soft placeholder. Dark mode fills with 30% of the input tone. Textareas and selects match.
- **Focus:** stroke turns to the ring colour and a solid 2px ring appears.
- **Error / Disabled:** invalid fields get an Alarm Red stroke and a 3px red ring at 20%; the message sits below in red label text. Disabled fields drop to 50% opacity on a 50% input tint with a not-allowed cursor.
- **Layout:** label above the field at label size, 8px gap, fields stacked with 16px to 20px between them; the submit button spans full width in the sign-in card and sits in a right-aligned row in page forms.

### Navigation
- **Style:** three items (Overview, Monitors, Incidents) at Nav size (1rem, 500) with 20px icons, 44px tall, 8px corners, 12px horizontal padding, 12px icon gap, 2px apart. The brand above is the lucide Activity icon in Signal Orange next to the wordmark "Nocu" at 600 weight, 1.125rem, with tight tracking, one step above the nav labels.
- **Default:** Ink Soft text. **Hover:** Warm Cream wash with Cocoa text. **Active:** Ink text plus a 3px-wide Signal Orange marker on the left edge of the sidebar, inset 8px from the item's top and bottom and rounded on its right side; the item itself has no fill. `aria-current="page"` carries the state.
- **Mobile:** a 56px top bar on Surface Sidebar with the hamburger, brand, and theme toggle; the hamburger opens a left sheet on Surface Sidebar holding the brand, the same nav, and the account menu in a hairline-topped footer. Icon buttons on these touch surfaces (hamburger, theme toggle, sheet close) are 44px square; the desktop rail keeps 32px.

### Tables
- **Style:** full-width, 0.875rem, header row 40px with 500-weight Ink text, cells 8px padded, hairline between rows, no outer border. Rows wash to 50% Surface Muted on hover. Row actions live in a ghost icon button that opens a dropdown. Long cells cap at `max-w-xs` or `max-w-md` and truncate.
- **Filters:** labelled selects in a row above the table; pagination below.

### Status Marks (signature)
Status never uses a filled panel; it uses a small mark next to text.
- **Status Dot:** 6px circle. Blue for up, orange for down, 40% Ink Soft for pending. Also used for event tone (blue up, orange down, 60% orange warn, grey info) and for the connection line (pulsing blue for "Live", solid orange for "Reconnecting…").
- **Severity Bar:** 2px-wide, fully rounded, stretches the row height in lists (or 16px inline next to its label). Alarm Red for critical in incident lists, Signal Orange for high, Clear Sky Blue for medium, 40% Ink Soft for low. The overview's compact bar uses orange for critical and 60% orange for high.
- **Heartbeat Strip:** the last 30 probes as 4px-wide bars (8px in the large size) with 2px gaps, oldest left, newest right. Height scales from 4px to 24px with latency relative to the slowest probe; colour is blue for up, orange for down, Surface Muted for pending or empty slots. Each bar carries a tooltip with time and latency, and the whole strip is an image with a plain-language label.

### Tooltip
Inverted: Ink surface with Paper text at label size, 8px corners, 12px by 6px padding, a small rotated square arrow, fading and scaling in from 95% over 100ms.

### Avatar
A 32px Warm Cream circle with the account's initial in Cocoa at body size and 500 weight. No photos.

### Empty, Loading, and Error States
Empty states are one grey body sentence in place of the rows ("Nothing needs attention right now.", "Events appear here as they happen."); list pages set it inside the same hairline-bordered panel the error state uses. Loading states are skeleton bars at the same heights as the content they replace, inside a region marked busy. Errors are a Hairline-bordered panel with a 500-weight title, a grey explanation, and an outline "Try again" button; inline load errors on a page header are a single line of Signal Orange body text.

## Do's and Don'ts

### Do:
- **Do** keep Signal Orange to primary actions, the active nav marker, focus rings, and down/high/critical status; count the orange on a calm screen and expect one marker and at most one button.
- **Do** use Clear Sky Blue for every "up" or healthy state and 40% Ink Soft for pending, idle, or low.
- **Do** carry status with a 6px dot, a 2px bar, or a pill badge next to plain text, never with a tinted row or card background.
- **Do** bound in-page surfaces with the 1px 10% ring or a Hairline border and reserve shadows for dropdowns, dialogs, and sheets.
- **Do** set URLs, timestamps, latency, and other machine strings in Geist Mono at label size in Ink Soft.
- **Do** keep controls at 32px tall with 10px corners and the shared solid 2px focus ring; step the corner down with the size.
- **Do** render a control the viewer cannot use as disabled with the "Needs the operator role" tooltip rather than hiding it.
- **Do** write one grey sentence for an empty state and use height-matched skeletons for loading.

### Don't:
- **Don't** introduce green, or any second saturated accent, for success, healthy, or online states.
- **Don't** set white text on an orange fill; the text on Signal Orange is always Ink.
- **Don't** fill a button, badge, or row with solid Alarm Red; red is a 10% tint with red text for destructive actions and a 2px bar for critical severity.
- **Don't** use orange on hover, icons at rest, headings, or decorative shapes.
- **Don't** add shadows, gradients, glows, or blurred backdrops to cards, tables, panels, or the sidebar.
- **Don't** use uppercase, letter-spaced, or eyebrow-style labels; section labels are sentence-case grey body text.
- **Don't** use Manrope above 600 weight or below 400, and don't set headings in Geist Mono.
- **Don't** wrap page-level lists or tables in cards; they sit directly on Paper with hairline dividers.
- **Don't** animate anything ambiently except the live-connection dot, and keep overlay transitions at 100ms to 200ms.
