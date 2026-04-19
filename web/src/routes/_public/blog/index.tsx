import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { motion } from 'framer-motion'
import { useInView } from 'react-intersection-observer'
import { getSeriesSlugs, blogSeries } from '~/lib/blog-series'

interface BlogFrontmatter {
  title: string
  pubDate: string
  description: string
  author?: string
  authorImage?: string
}

interface BlogModule {
  frontmatter: BlogFrontmatter
  default: React.ComponentType
}

const modules = import.meta.glob<BlogModule>('/src/content/blog/consulting/*.{md,mdx}', {
  eager: true,
})

function getSlug(path: string) {
  const filename = path.split('/').pop() || ''
  return filename.replace(/\.(md|mdx)$/, '')
}

interface PostEntry {
  kind: 'post'
  slug: string
  title: string
  pubDate: string
  description: string
  author?: string
  authorImage?: string
}

interface SeriesEntry {
  kind: 'series'
  id: string
  title: string
  subtitle: string
  author: string
  authorImage: string
  pubDate: string
  latestPubDate: string
  publishedCount: number
  totalCount: number
}

type FeedEntry = PostEntry | SeriesEntry

function getFeed(): FeedEntry[] {
  const now = new Date()
  const seriesSlugs = getSeriesSlugs()

  const posts: FeedEntry[] = Object.entries(modules)
    .map(([path, mod]) => ({
      kind: 'post' as const,
      slug: getSlug(path),
      ...mod.frontmatter,
    }))
    .filter((post) => new Date(post.pubDate) <= now && !seriesSlugs.has(post.slug))

  const seriesCards: FeedEntry[] = blogSeries
    .map((s) => {
      const publishedDates: Date[] = []
      for (const ch of s.chapters) {
        for (const [path, mod] of Object.entries(modules)) {
          if (getSlug(path) === ch.slug && new Date(mod.frontmatter.pubDate) <= now) {
            publishedDates.push(new Date(mod.frontmatter.pubDate))
          }
        }
      }
      if (publishedDates.length === 0) return null
      const earliest = publishedDates.reduce((a, b) => (a < b ? a : b))
      const latest = publishedDates.reduce((a, b) => (a > b ? a : b))
      return {
        kind: 'series' as const,
        id: s.id,
        title: s.title,
        subtitle: s.subtitle,
        author: s.author,
        authorImage: s.authorImage,
        pubDate: earliest.toISOString(),
        latestPubDate: latest.toISOString(),
        publishedCount: publishedDates.length,
        totalCount: s.chapters.length,
      }
    })
    .filter((x): x is SeriesEntry => x !== null)

  return [...posts, ...seriesCards].sort(
    (a, b) => new Date(b.pubDate).valueOf() - new Date(a.pubDate).valueOf()
  )
}

function FadeIn({ children, className }: { children: React.ReactNode; className?: string }) {
  const { ref, inView } = useInView({ triggerOnce: true, threshold: 0.1 })
  return (
    <motion.div
      ref={ref}
      initial={{ opacity: 0, y: 20 }}
      animate={inView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
      className={className}
    >
      {children}
    </motion.div>
  )
}

export const Route = createFileRoute('/_public/blog/')({
  component: ConsultingBlogIndex,
})

function ConsultingBlogIndex() {
  const feed = getFeed()
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <div className="font-inter min-h-screen bg-stone-50">
      {/* Nav */}
      <header className="px-4 pt-8 pb-0 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <div className="mb-0 flex items-center justify-between">
            <Link
              to="/"
              className="font-logo text-2xl tracking-tight text-stone-900 no-underline hover:no-underline md:text-3xl"
            >
              🦌 <span className="text-green-700">deer.sh</span>
            </Link>
            <div className="hidden items-center gap-6 text-sm text-stone-500 md:flex">
              <Link to="/" className="transition-colors hover:text-stone-800">
                Home
              </Link>
              <a href="/#services" className="transition-colors hover:text-stone-800">
                Services
              </a>
              <Link to="/product" className="transition-colors hover:text-stone-800">
                Product
              </Link>
              <span className="font-medium text-stone-800">Blog</span>
              <a
                href="/#contact"
                className="inline-flex items-center gap-1 rounded-full border border-green-900/40 bg-green-900/10 px-4 py-1.5 text-green-700 transition-colors hover:border-green-900/60 hover:bg-green-900/20"
              >
                Get in Touch
              </a>
            </div>
            <button
              className="text-stone-500 hover:text-stone-800 md:hidden"
              onClick={() => setMobileOpen(!mobileOpen)}
            >
              {mobileOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </button>
          </div>

          {mobileOpen && (
            <div
              className="fixed inset-0 z-30 bg-stone-900 md:hidden"
              onClick={() => setMobileOpen(false)}
            >
              <nav
                className="flex flex-col gap-6 p-8 pt-20 text-lg text-stone-300"
                onClick={(e) => e.stopPropagation()}
              >
                <Link
                  to="/"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Home
                </Link>
                <a
                  href="/#services"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Services
                </a>
                <Link
                  to="/product"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Product
                </Link>
                <a
                  href="/#contact"
                  onClick={() => setMobileOpen(false)}
                  className="transition-colors hover:text-white"
                >
                  Get in Touch
                </a>
              </nav>
            </div>
          )}
        </div>
      </header>

      {/* Header */}
      <section className="px-4 pt-16 pb-8 sm:px-6">
        <div className="mx-auto max-w-2xl">
          <motion.div
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
          >
            <h1 className="font-logo text-3xl font-bold tracking-tight text-stone-900 md:text-4xl">
              Blog
            </h1>
            <p className="mt-4 leading-relaxed text-stone-700">
              Technical deep dives, engineering notes, and ELK stack insights from our consulting
              practice.
            </p>
          </motion.div>
        </div>
      </section>

      {/* Feed */}
      <main className="px-4 pb-24 sm:px-6">
        <div className="mx-auto max-w-2xl space-y-3">
          {feed.map((entry) =>
            entry.kind === 'series' ? (
              <FadeIn key={`series-${entry.id}`}>
                <Link
                  to="/blog/series/$seriesId"
                  params={{ seriesId: entry.id }}
                  className="group block rounded-2xl border border-green-900/20 bg-white p-5 no-underline transition-all duration-200 hover:-translate-y-0.5 hover:border-green-900/40 hover:no-underline"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-xs font-medium tracking-wide text-green-700 uppercase">
                        Technical Deep Dive
                      </span>
                      <span className="rounded-full bg-green-900/10 px-2 py-0.5 text-xs text-green-700">
                        {entry.publishedCount}/{entry.totalCount} parts
                      </span>
                    </div>
                    <time
                      dateTime={new Date(entry.latestPubDate).toISOString()}
                      className="text-xs text-stone-400"
                    >
                      {new Date(entry.latestPubDate).toLocaleDateString('en-us', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </time>
                  </div>
                  <h2 className="mt-2 text-sm font-semibold text-stone-900 transition-colors group-hover:text-green-700">
                    {entry.title}
                  </h2>
                  <p className="mt-1 text-sm text-stone-500">{entry.subtitle}</p>
                  <div className="mt-3 flex items-center gap-2">
                    <img
                      src={entry.authorImage}
                      alt={entry.author}
                      className="h-5 w-5 rounded-full border border-stone-200"
                    />
                    <span className="text-xs text-stone-400">{entry.author}</span>
                  </div>
                </Link>
              </FadeIn>
            ) : (
              <FadeIn key={entry.slug}>
                <Link
                  to="/blog/$slug"
                  params={{ slug: entry.slug }}
                  className="group block rounded-2xl border border-stone-200 bg-white p-5 no-underline transition-all duration-200 hover:-translate-y-0.5 hover:border-green-900/30 hover:no-underline"
                >
                  <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                    <h2 className="text-sm font-semibold text-stone-900 transition-colors group-hover:text-green-700">
                      {entry.title}
                    </h2>
                    <time
                      dateTime={new Date(entry.pubDate).toISOString()}
                      className="text-xs whitespace-nowrap text-stone-400"
                    >
                      {new Date(entry.pubDate).toLocaleDateString('en-us', {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </time>
                  </div>
                  <p className="mt-2 line-clamp-2 text-sm text-stone-500">{entry.description}</p>
                  {entry.author && (
                    <div className="mt-3 flex items-center gap-2">
                      {entry.authorImage ? (
                        <img
                          src={entry.authorImage}
                          alt={entry.author}
                          className="h-5 w-5 rounded-full border border-stone-200"
                        />
                      ) : (
                        <div className="flex h-5 w-5 items-center justify-center rounded-full border border-stone-200 bg-stone-100 text-xs text-green-700">
                          {entry.author.charAt(0)}
                        </div>
                      )}
                      <span className="text-xs text-stone-400">{entry.author}</span>
                    </div>
                  )}
                </Link>
              </FadeIn>
            )
          )}
        </div>
      </main>
    </div>
  )
}
