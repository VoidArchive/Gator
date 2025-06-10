# Gator

A command-line RSS feed aggregator built in Go. Gator allows you to manage RSS feeds, follow feeds from other users, and browse recent posts from your followed feeds right in your terminal.

## Features

- 🔐 **User Management** - Register users and switch between accounts
- 📡 **RSS Feed Management** - Add, list, and manage RSS feeds
- 👥 **Social Following** - Follow feeds created by other users
- 🤖 **Auto-Aggregation** - Continuous RSS feed scraping and post storage
- 📰 **Post Browsing** - View recent posts from your followed feeds
- 🗄️ **Database Storage** - Persistent storage of users, feeds, and posts

## Prerequisites

Before installing Gator, make sure you have:

- **Go 1.20+** - [Download and install Go](https://golang.org/dl/)
- **PostgreSQL** - [Download and install PostgreSQL](https://www.postgresql.org/download/)

## Installation

Install Gator using Go's built-in package manager:

```bash
go install github.com/voidarchive/Gator@latest
```

This will compile and install the `gator` binary to your `$GOPATH/bin` directory (usually `~/go/bin`).

Make sure your `$GOPATH/bin` is in your system's `$PATH` so you can run `gator` from anywhere.

## Setup

### 1. Database Setup

Create a PostgreSQL database for Gator:

```sql
CREATE DATABASE gator;
```

### 2. Run Database Migrations

Since this project doesn't include an automated migration system, you'll need to manually create the database schema. Run these SQL commands in your PostgreSQL database:

```sql
-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL UNIQUE
);

-- Feeds table
CREATE TABLE feeds (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    last_fetched_at TIMESTAMP
);

-- Feed follows table
CREATE TABLE feed_follows (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feed_id UUID NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
    UNIQUE (user_id, feed_id)
);

-- Posts table
CREATE TABLE posts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    description TEXT,
    published_at TIMESTAMP,
    feed_id UUID NOT NULL REFERENCES feeds (id) ON DELETE CASCADE
);
```

### 3. Configuration File

Create a configuration file at `~/.gatorconfig.json`:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

Replace `username`, `password`, and database connection details with your PostgreSQL credentials.

## Usage

### User Management

```bash
# Register a new user
gator register <username>

# Login as a user (sets current user)
gator login <username>

# List all users
gator users

# Reset database (delete all users)
gator reset
```

### Feed Management

```bash
# Add a new RSS feed (automatically follows it)
gator addfeed "Feed Name" "https://example.com/rss"

# List all feeds in the system
gator feeds

# Follow an existing feed by URL
gator follow "https://example.com/rss"

# List feeds you're following
gator following

# Unfollow a feed
gator unfollow "https://example.com/rss"
```

### Post Aggregation & Browsing

```bash
# Start RSS aggregation (runs continuously)
gator agg <duration>

# Examples:
gator agg 1m     # Fetch feeds every 1 minute
gator agg 30s    # Fetch feeds every 30 seconds
gator agg 1h     # Fetch feeds every 1 hour

# Browse recent posts from followed feeds
gator browse          # Show 2 most recent posts (default)
gator browse 10       # Show 10 most recent posts
```

## Example Workflow

```bash
# 1. Register and login
gator register alice
gator login alice

# 2. Add some RSS feeds
gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"

# 3. Start aggregating posts (in background)
gator agg 5m &

# 4. Browse recent posts
gator browse 5

# 5. Follow feeds from other users
gator follow "https://blog.boot.dev/index.xml"
```

## Popular RSS Feeds to Try

Here are some popular RSS feeds you can add to get started:

- **TechCrunch**: `https://techcrunch.com/feed/`
- **Hacker News**: `https://news.ycombinator.com/rss`
- **Boot.dev Blog**: `https://blog.boot.dev/index.xml`
- **GitHub Blog**: `https://github.blog/feed/`
- **Go Blog**: `https://go.dev/blog/feed.atom`

## Development

To run Gator in development mode:

```bash
git clone https://github.com/voidarchive/Gator.git
cd Gator
go run . <command> [args...]
```

To build a binary:

```bash
go build -o gator
./gator <command> [args...]
```

## Architecture

Gator is built with:

- **Go** - Backend logic and CLI interface
- **PostgreSQL** - Data persistence
- **SQLC** - Type-safe SQL code generation
- **RSS/XML Parsing** - Built-in Go XML parsing

The application follows a clean architecture with separate packages for:
- `internal/cli` - Command handlers and CLI logic
- `internal/config` - Configuration management
- `internal/database` - Database queries and models
- `sql/` - SQL schema and queries

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the [MIT License](LICENSE).
