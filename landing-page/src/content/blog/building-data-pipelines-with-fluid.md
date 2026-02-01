---
title: "Building Data Pipelines with Fluid"
pubDate: 2026-01-24
description: "How I built a HN search app with Fluid"
author: "Collin @ Fluid.sh"
authorImage: "../images/skeleton_smoking_cigarette.jpg"
authorEmail: "cpfeifer@madcactus.org"
authorPhone: "+3179955114"
authorDiscord: "https://discordapp.com/users/301068417685913600"
---

## Intro

_Disclaimer: I build a lot of data pipelines_. After doing it a while, it get's quite mundane. Get the data source, see what the data looks like, see what the data needs to be and find the delta.

A large part of what you do as a data engineer is being a middleman. Talking to customers (or coworkers) about what data they want, how they want it formatted, what they have built around it in the past. Then the engineer applies good old engineering glue to get everything working right. Pipelines that know where the data comes from, what it looks like, schedules, timing etc. Until it doesn't.

I wanted to see if Fluid could help me in my endeavor to make this process not suck.

## Data

First I needed some data. Preferrably something that could be searched/indexed easy and not be too much data to make me lose an arm and a leg on hosting.

So I went for the logs from the https://fluid.sh website! Easily indexable, text search is not too expensive and fun to search through the logs.

Naturally I went for what I'm used to, ELK stack.

## Ingesting

Next the hard part. Ingesting this data from the Astro to Elasticsearch. I was feeling lazy and I brought out Fluid to do it for me.

I decided to use Logstash for efficiency and simplicity sake.

The landing page is currently being hosted in Render. Render provides a service to forward logs from a Render service to the Logstash that Fluid will set up.

First we setup some middleware in Astro.

```typescript
import type { MiddlewareNext, APIContext } from "astro";
export const onRequest = async (
  { request }: APIContext,
  next: MiddlewareNext,
) => {
  const start = Date.now();
  const response = await next();
  console.log(
    JSON.stringify({
      type: "http_access",
      method: request.method,
      path: new URL(request.url).pathname,
      status: response.status,
      duration_ms: Date.now() - start,
    }),
  );
  return response;
};
```

After that we turn we now seeing logs coming from the site.

```json
{
  "type": "http_access",
  "method": "GET",
  "path": "/",
  "status": 200,
  "duration_ms": 3
}
```

Here is the prompt use for Fluid

```
Hey can you create a sandbox of test-vm-1 and
install logstash and ingest data that will
be coming from http? Here is the data example:
  {
    "type":"http_access",
    "method":"GET",
    "path":"/",
    "status":200,
    "duration_ms":3
  }
  I would like the resulting data from the
  API to be in JSON format, and outputted to an
  elasticsearch instance
```
