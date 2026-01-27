import { defineCollection, z } from "astro:content";

const blog = defineCollection({
  type: "content",
  // Type-check frontmatter using a schema
  schema: z.object({
    title: z.string(),
    description: z.string(),
    // Transform string to Date object
    pubDate: z.coerce.date(),
    updatedDate: z.coerce.date().optional(),
    heroImage: z.string().optional(),
    // Author info
    author: z.string().optional(),
    authorImage: z.string().optional(),
    authorEmail: z.string().optional(),
    authorPhone: z.string().optional(),
    authorDiscord: z.string().optional(),
  }),
});

export const collections = { blog };
