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
  - [API Routes](#api-routes)
  - [Core Logic](#core-logic)
  - [Sample API Requests and Responses](#sample-api-requests-and-responses)
      - [1. Fetch Repository Details](#1-fetch-repository-details)
      - [2. Fetch Commits for a Repository](#2-fetch-commits-for-a-repository)
      - [3. Reset Collection](#3-reset-collection)
      - [4. Get Top Authors](#4-get-top-authors)
      - [5. Monitor a Repository](#5-monitor-a-repository)
  - [Background Task: Fetching Data at Short Intervals](#background-task-fetching-data-at-short-intervals)
  - [Running Tests](#running-tests)

---

## Download and Run the App

### Prerequisites

- **Git:** Installed on your machine.
- **Docker & Docker Compose:** Installed and running.
- **Go:** Version 1.23 or later.
- **direnv:** Version 2.35.0
- **make:** Version 3.81

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
make seed-live
```
This script seeds your database with Chromium-specific data.

---

## How to Use the App

Lema exposes majorly four endpoints. You can use your favorite HTTP client (e.g., browser, Postman, curl) to interact with these endpoints.

## API Routes

The following routes are available in the application:

- **GET /v1/repositories/{repository_name}/commits?owner_name={owner_name}** - Get commits for a repository.
- **GET /v1/commit-authors/top?limit=10** - Get top authors by commit count.
- **POST /v1/repositories/reset-collection** - Reset the collection of a repository.
- **POST /v1/repositories/monitor** - Add a new repository to the monitoring list.

## Core Logic

The core logic of the application is primarily located in the `internal` and `internal/services` directories. The `services` package contains business logic related to repositories, commits and GitHub interactions. For the monitoring part, I made use of a package called `Asynq` which checks for the repositories and then make a call to get the latest changes. To change / update the frequency when checking, you can make use of the `cron.yml` file.

## Sample API Requests and Responses

#### 1. Fetch Repository Details
**Method**: GET  
**URL**: `http://localhost:3000/v1/repositories/chromium?owner_name=chromium`  
**Sample Response**:
```json
{
    "name": "chromium",
    "description": "The official GitHub mirror of the Chromium source",
    "url": "https://api.github.com/repos/chromium/chromium",
    "language": "C++",
    "forks_count": 6805,
    "stargazers_count": 18393,
    "open_issues_count": 93,
    "watchers_count": 18393,
    "created_at": "2018-02-05T20:55:32Z",
    "updated_at": "2024-08-04T03:16:04Z"
}
```

#### 2. Fetch Commits for a Repository
**Method**: GET  
**URL**: `http://localhost:3000/v1/repos/chromium/commits?owner_name=chromium&page=1&page_size=50`  
**Sample Response**:
```json
{
    "status": true,
    "data": {
        "pagination": {
            "page": 1,
            "page_size": 10,
            "total_pages": 10,
            "total_items": 100
        },
        "data": [
            {
                "id": "c0642115-8377-400f-bd5f-eecbeca3a010",
                "sha": "642a15ef9175bb2f84067dfad3363fc41a229e26",
                "url": "https://api.github.com/repos/chromium/chromium/commits/642a15ef9175bb2f84067dfad3363fc41a229e26",
                "message": "Roll ANGLE from d29b8e05185b to fac33bb32383 (1 revision)\n\nhttps://chromium.googlesource.com/angle/angle.git/+log/d29b8e05185b..fac33bb32383\n\n2025-03-18 shufen.ma@arm.com Refine InterfaceVariablesMatch\n\nIf this roll has caused a breakage, revert this CL and stop the roller\nusing the controls here:\nhttps://autoroll.skia.org/r/angle-chromium-autoroll\nPlease CC angle-team@google.com,syoussefi@google.com on the revert to ensure that a human\nis aware of the problem.\n\nTo file a bug in ANGLE: https://bugs.chromium.org/p/angleproject/issues/entry\nTo file a bug in Chromium: https://bugs.chromium.org/p/chromium/issues/entry\n\nTo report a problem with the AutoRoller itself, please file a bug:\nhttps://issues.skia.org/issues/new?component=1389291&template=1850622\n\nDocumentation for the AutoRoller is here:\nhttps://skia.googlesource.com/buildbot/+doc/main/autoroll/README.md\n\nCq-Include-Trybots: luci.chromium.try:android_optional_gpu_tests_rel;luci.chromium.try:linux_optional_gpu_tests_rel;luci.chromium.try:mac_optional_gpu_tests_rel;luci.chromium.try:win_optional_gpu_tests_rel;luci.chromium.try:linux-swangle-try-x64;luci.chromium.try:win-swangle-try-x86\nBug: None\nTbr: syoussefi@google.com\nChange-Id: I2ed8c2cfef323bdf6b0ee8d7aeae0374370b3d7e\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/6367646\nCommit-Queue: chromium-autoroll <chromium-autoroll@skia-public.iam.gserviceaccount.com>\nBot-Commit: chromium-autoroll <chromium-autoroll@skia-public.iam.gserviceaccount.com>\nCr-Commit-Position: refs/heads/main@{#1434138}",
                "date": "2025-03-18T15:14:38Z",
                "repository": {
                    "id": "f241a429-efc3-4df2-94fd-73e5571636c3",
                    "name": "chromium",
                    "owner_name": "chromium",
                    "description": "The official GitHub mirror of the Chromium source",
                    "url": "https://api.github.com/repos/chromium/chromium",
                    "language": "C++",
                    "forks_count": 7428,
                    "stars_count": 20155,
                    "watchers_count": 20155,
                    "open_issues_count": 118
                },
                "author": {
                    "id": "40ab3934-5ad0-4b54-ae10-499cb9d3d87e",
                    "name": "chromium-autoroll",
                    "email": "chromium-autoroll@skia-public.iam.gserviceaccount.com"
                }
            },
            {
                "id": "4e059d5e-513f-4ec9-8c00-4edf3dee0084",
                "sha": "fe92421aa73f0d4c6e72e996c2eeb99d84928ef2",
                "url": "https://api.github.com/repos/chromium/chromium/commits/fe92421aa73f0d4c6e72e996c2eeb99d84928ef2",
                "message": "[Tab Groups] Remove getRelatedTabCountForRootId\n\n- getRelatedTabCountForRootId is replaced by getTabCountForGroup\n  remove it.\n- Update outdated javadoc.\n- Add missing @Nullable.\n\nBug: 399354986\nChange-Id: I29c5e2339aba5caac6478838a26325674733c32e\nReviewed-on: https://chromium-review.googlesource.com/c/chromium/src/+/6367574\nReviewed-by: Dan Polanco <polardz@google.com>\nCommit-Queue: Calder Kitagawa <ckitagawa@chromium.org>\nCr-Commit-Position: refs/heads/main@{#1434137}",
                "date": "2025-03-18T15:07:42Z",
                "repository": {
                    "id": "f241a429-efc3-4df2-94fd-73e5571636c3",
                    "name": "chromium",
                    "owner_name": "chromium",
                    "description": "The official GitHub mirror of the Chromium source",
                    "url": "https://api.github.com/repos/chromium/chromium",
                    "language": "C++",
                    "forks_count": 7428,
                    "stars_count": 20155,
                    "watchers_count": 20155,
                    "open_issues_count": 118
                },
                "author": {
                    "id": "4c8b958c-e7c5-47c4-8c7c-dff1ae6e9fd3",
                    "name": "Calder Kitagawa",
                    "email": "ckitagawa@chromium.org"
                }
            }
        ]
    },
    "message": "Commits stored and retrieved successfully"
}
```

#### 3. Reset Collection
**Method**: POST  
**URL**: `http://localhost:3000/v1/repositories/reset-collection`  
<br>
**Sample Request**:
```json
{
    "repo_name": "chromium",
    "owner_name": "chromium",
    "start_time": "2025-03-20T01:30:00Z"
}
```
**Sample Response**:
```json
{
    "status": true,
    "message": "Reset commits started for repository named chromium/chromium"
}
```

#### 4. Get Top Authors
**Method**: GET  
**URL**: `http://localhost:3000/v1/commit-authors/top?limit=3`  
**Sample Response**:
```json
{
    "status": true,
    "data": [
        {
            "id": "40ab3934-5ad0-4b54-ae10-499cb9d3d87e",
            "name": "chromium-autoroll",
            "email": "chromium-autoroll@skia-public.iam.gserviceaccount.com",
            "commit_count": 21
        },
        {
            "id": "325dfede-f810-4400-8079-3e7dec6cad51",
            "name": "chromium-internal-autoroll",
            "email": "chromium-internal-autoroll@skia-corp.google.com.iam.gserviceaccount.com",
            "commit_count": 6
        },
        {
            "id": "4c8b958c-e7c5-47c4-8c7c-dff1ae6e9fd3",
            "name": "Calder Kitagawa",
            "email": "ckitagawa@chromium.org",
            "commit_count": 4
        }
    ],
    "message": "top author commits retrieved successfully"
}
```

#### 5. Monitor a Repository
**Method**: POST  
**URL**: `http://localhost:300/api/repositories/monitor` 
<br>
**Sample Request**:
```json
{
    "repo_name": "chromium",
    "owner_name": "chromium",
    "start_time": "2025-03-20T01:30:00Z"
}
``` 
**Sample Response**:
```json
{
    "status": true,
    "message": "Monitoring started for repository named babyfaceeasy/myresume"
}
```

These examples showcase how to interact with the LEMA API. The endpoints support various actions such as fetching repository details, commit history, resetting data collections, and starting monitoring for repositories.

---

## Background Task: Fetching Data at Short Intervals

Lema includes a background task that updates the commit data for all repositories in the database. By default, this task runs every 10 minutes, but you can modify the schedule:
- The schedule is defined in the `cron.yml` file using standard cron syntax.
- You can edit the `cron.yml` file while the app is running to adjust the frequency.

----

## Running Tests

A test exists for the GET TOP N AUTHORS BY COMMITS endpoint. To run all tests, execute:
```bash
go test ./...
```