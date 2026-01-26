# Landing Page Development Guidelines

## Tech Stack

- **Framework**: Astro 5
- **Styling**: Tailwind CSS v4
- **Package Manager**: Bun
- **Analytics**: PostHog

## Scripts

```bash
# Install dependencies
bun install

# Development server (hot reload)
bun run dev

# Build for production
bun run build

# Preview production build
bun run preview
```

## Project Structure

```
landing-page/
├── src/
│   ├── components/     # Astro components
│   ├── content/        # Content collections (blog)
│   ├── layouts/        # Base layouts
│   ├── pages/          # Routes (index.astro, blog/, etc.)
│   └── styles/         # Global styles
├── public/             # Static assets
├── package.json
└── astro.config.mjs
```

## Design System

### Color Palette

Terminal-inspired dark theme with blue accents:

| Purpose          | Color       | Tailwind Class                             |
| ---------------- | ----------- | ------------------------------------------ |
| Background       | Black       | `bg-black`                                 |
| Card/Surface     | Dark gray   | `bg-neutral-900`, `bg-neutral-900/50`      |
| Primary text     | Light gray  | `text-neutral-200`, `text-neutral-300`     |
| Secondary text   | Muted gray  | `text-neutral-400`, `text-neutral-500`     |
| Accent           | Blue        | `text-blue-400`, `border-blue-400`         |
| Success          | Green       | `text-green-400`, `border-green-400`       |
| Warning/Playbook | Amber       | `text-amber-400`, `border-amber-400`       |
| Borders          | Subtle gray | `border-neutral-800`, `border-neutral-700` |

### Typography

- **Logo/Brand**: `font-logo` (custom)
- **Code/Terminal**: `font-mono`
- **Body**: Default sans-serif

### Component Patterns

#### Cards

```html
<div
  class="p-4 rounded-lg bg-neutral-900/50 border border-neutral-800 hover:border-blue-500/30 transition-all duration-300"
>
  <!-- content -->
</div>
```

#### Terminal Blocks

```html
<div
  class="px-4 py-3 font-mono text-sm rounded-lg bg-neutral-900 text-neutral-300"
>
  <span class="text-blue-400">$</span> command here
</div>
```

#### Timeline Items (with glow effect)

```html
<div
  class="timeline-dot w-2 h-2 rounded-full bg-blue-400 shadow-[0_0_8px_2px_rgba(96,165,250,0.6)]"
></div>
```

#### Glowing Borders

```css
/* Blue glow */
shadow-[0_0_8px_2px_rgba(96,165,250,0.6)]

/* Green glow */
shadow-[0_0_8px_2px_rgba(74,222,128,0.6)]

/* Amber glow */
shadow-[0_0_8px_2px_rgba(251,191,36,0.5)]
```

### Animation Patterns

#### Intersection Observer for scroll-triggered animations

```javascript
const observer = new IntersectionObserver(
  (entries) => {
    entries.forEach((entry) => {
      if (entry.isIntersecting) {
        // Trigger animation
        observer.disconnect();
      }
    });
  },
  { threshold: 0.3 },
);
```

#### Sequential item reveal

```javascript
items.forEach((item, index) => {
  setTimeout(() => {
    item.classList.remove("opacity-0", "translate-y-4");
    item.classList.add("opacity-100", "translate-y-0");
  }, index * 250);
});
```

#### Typing animation with mistakes

- Use array of `['type', char]`, `['delete', '']`, `['pause', ms]` actions
- Variable delay: `30 + Math.random() * 40` ms between keystrokes
- Faster deletion: `50ms` per character

### Icon Style

Terminal-inspired text icons using monospace font:

| Concept      | Icon    |
| ------------ | ------- |
| Home/Sandbox | `[~]`   |
| List/Explore | `ls`    |
| Output/Audit | `>>>`   |
| YAML/Config  | `.yaml` |
| Prompt       | `$`     |
| Checkmark    | `v`     |

## Key Pages

| Route         | File                             | Purpose                 |
| ------------- | -------------------------------- | ----------------------- |
| `/`           | `src/pages/index.astro`          | Main landing page       |
| `/install.sh` | `src/pages/install.sh.ts`        | Install script endpoint |
| `/blog/*`     | `src/pages/blog/[...slug].astro` | Blog posts              |

## Development Notes

- Use `<script>` tags in Astro components for client-side JS
- Animations trigger on scroll via IntersectionObserver
- No React needed - pure Astro + vanilla JS
- Tailwind v4 uses new config format (CSS-based)
