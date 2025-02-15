# Blog Aggregator CLI (gator)

## Overview
`gator` is a command-line tool for aggregating and browsing RSS feeds. It allows users to register, log in, follow feeds, browse posts, and more.

## Prerequisites
Before using `gator`, ensure you have the following installed:

- [Go](https://go.dev/dl/) (1.21 or later)
- [PostgreSQL](https://www.postgresql.org/download/)

## Installation
To install `gator`, run:
```sh
go install github.com/SethGK/blog_aggregator@latest
```
This will place the `gator` binary in your `$GOPATH/bin`. Ensure `$GOPATH/bin` is in your system `PATH` to run it globally.

## Database Setup
1. Start PostgreSQL and create a new database:
   ```sh
   createdb blog_aggregator
   ```
2. Apply the database migrations (if applicable) using your preferred migration tool.

## Configuration
`gator` requires a configuration file to connect to the database. Create a `.env` file in the root directory:
```sh
DATABASE_URL=postgres://user:password@localhost:5432/blog_aggregator?sslmode=disable
```
Replace `user` and `password` with your PostgreSQL credentials.

## Running the Program
For development, run:
```sh
go run .
```
For production, simply use:
```sh
gator
```

## Usage
Here are all available commands:

### User Management
- **Register a new user:**
  ```sh
  gator register
  ```
  Prompts for email and password to create an account.

- **Login:**
  ```sh
  gator login
  ```
  Authenticates a user and starts a session.

- **Reset database:**
  ```sh
  gator reset
  ```
  Resets all users in the database.

- **List all users:**
  ```sh
  gator users
  ```
  Displays all registered users.

### Feed Management
- **Add a new RSS feed:**
  ```sh
  gator addfeed [feed_url]
  ```
  Adds a new RSS feed to the system.

- **List all available feeds:**
  ```sh
  gator feeds
  ```
  Displays all available feeds in the system.

- **Follow a feed:**
  ```sh
  gator follow [feed_id]
  ```
  Subscribes the user to a specific RSS feed.

- **List followed feeds:**
  ```sh
  gator following
  ```
  Displays all feeds the user is currently following.

- **Unfollow a feed:**
  ```sh
  gator unfollow [feed_id]
  ```
  Removes a feed from the user's followed list.

### Browsing Posts
- **Browse latest posts:**
  ```sh
  gator browse [limit]
  ```
  Displays the latest posts from followed feeds. Default limit is `2`.

- **Aggregate new posts:**
  ```sh
  gator agg
  ```
  Fetches new posts from all subscribed feeds and updates the database.