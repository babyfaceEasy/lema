# Lema

Lema is an application that provides commit data from public repositories. This README explains how to download and run the app, seed the database, use the API endpoints, configure background data updates, and run tests.

---

## Table of Contents

- [Lema](#lema)
  - [Table of Contents](#table-of-contents)
  - [Download and Run the App](#download-and-run-the-app)
    - [Prerequisites](#prerequisites)
    - [Steps](#steps)
  - [Seed the Database](#seed-the-database)
  - [How to Use the App](#how-to-use-the-app)
    - [GET TOP N AUTHORS BY COMMITS](#get-top-n-authors-by-commits)
    - [GET REPOSITORY COMMITS](#get-repository-commits)
    - [Background Task: Fetching Data at Short Intervals](#background-task-fetching-data-at-short-intervals)
    - [Running Tests](#running-tests)

---

## Download and Run the App

### Prerequisites

- **Git:** Installed on your machine.
- **Docker & Docker Compose:** Installed and running.
- **Go:** Version 1.23 or later.
- **direnv:** Version 2.35.0

### Steps

1. **Clone the Repository**

   ```bash
   git clone https://github.com/babyfaceEasy/lema.git
   cd lema
   ```
2. **Load environment variables** <br>
   Copy the content of `.envrc.example` into `.envrc`
    ```bash
    cp .envrc.example .envrc
    direnv allow
    ```
3. **Run the Backing Services**
    ```bash
    docker compose -f docker/docker-compose.yaml up -d --build
    ```
4. **Install Dependencies and Run the App**
    ```bash
    go mod tidy
    go run cmd/apiserver/main.go
    ```
    This will install the needed dependencies and start the API server.

---

## Seed the Database

To populate the database with Chromium data, run the following command:
```bash
./seed_db.sh chromium
```
This script seeds your database with Chromium-specific data.

---

## How to Use the App

Lema exposes two main endpoints. You can use your favorite HTTP client (e.g., browser, Postman, curl) to interact with these endpoints.

### GET TOP N AUTHORS BY COMMITS

- **URL:** `http://localhost:3000/v1/commit-authors/top` <br>
- **Method:** GET <br>
- **Query Parameter:** limit (an integer that determines how many authors to return; defaults to 10 if not provided)

Example:
```bash
http://localhost:3000/v1/commit-authors/top?limit=5
```
This endpoint returns the top N authors by commit count.

### GET REPOSITORY COMMITS
- **URL:** `http://localhost:3000/v1/repositories/:repository_name/commits` <br>
- **Method:** GET
- **Path Parameter:** `repository_name` (the name of the public repository)
- **Query Parameter:** `until`
- - Only commits before this date are returned
- - Expected format: `YYYY-MM-DDTHH:MM:SSZ`
- - Example: `2024-01-12T10:30:00Z`


- **Query Parameter:** `owner_name`
- - This is the name of the owner of the repository.
- - Expected format: `string`
- - Example: `chromium`
- - Defaults to the path parameter `repository_name` if not provided

Example:
```bash
http://localhost:3000/v1/repositories/chromium/commits?until=2024-01-12T10:30:00Z&owner_name=chromium
```

---

### Background Task: Fetching Data at Short Intervals
---
Lema includes a background task that updates the commit data for all repositories in the database. By default, this task runs every 10 minutes, but you can modify the schedule:
- The schedule is defined in the `cron.yml` file using standard cron syntax.
- You can edit the `cron.yml` file while the app is running to adjust the frequency.

----

### Running Tests
---
A test exists for the GET TOP N AUTHORS BY COMMITS endpoint. To run all tests, execute:
```bash
go test ./...
```