# 🚨 HackerOne Alert Bot

**Stay ahead of the hunt.** A lightweight Go bot that monitors HackerOne's entire bug bounty directory and alerts you the moment a new program launches or an existing program updates its scope - so you can be among the first to hack on fresh targets.

## Why I Built This

As a bug bounty hunter, timing matters. New programs and scope changes mean fresh attack surface before the crowd arrives. Manually refreshing HackerOne's directory isn't scalable, so I automated it.

This bot scrapes HackerOne's internal GraphQL API, diffs the results against a local database, and fires off an email alert with the list of changed programs - fully hands-free.

## Architecture

```
HackerOne GraphQL API
        │
        │  Concurrent paginated fetches (goroutines)
        ▼
┌─────────────────┐
│   Go Workers    │──► Fan-out fetch, fan-in via channels
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    MongoDB      │──► Diff: new programs & scope updates
└────────┬────────┘
         │  Changed program handles
         ▼
┌─────────────────┐
│  Gmail SMTP     │──► HTML alert email
└─────────────────┘
```

### How It Works

1. **Discover** - Queries HackerOne's GraphQL endpoint to get the total program count, then fans out concurrent goroutines to fetch all programs (24 per page).
2. **Diff** - Each fetched program is compared against MongoDB by `handle`. New programs are inserted; programs with a newer `last_updated_at` timestamp are flagged and replaced.
3. **Alert** - All flagged program handles are collected and sent as a single HTML email via Gmail SMTP.

> On the first run, it seeds the database. On every run after, it diffs and alerts.

## Setup

```env
# .env
MONGODB_URI=mongodb://localhost:27017
FROM_EMAIL=your-email@gmail.com
FROM_EMAIL_PASSWORD=your-app-password
TO_EMAIL=recipient@example.com
```

```bash
go run .
```

Schedule with **cron** or **GitHub Actions** for continuous monitoring.

## Tech Stack

Go · MongoDB · HackerOne GraphQL · Gmail SMTP · gjson · godotenv
