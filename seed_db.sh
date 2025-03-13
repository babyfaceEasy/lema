#!/bin/bash

# Check if repository name is provided as an argument.
if [ -z "$1" ]; then
    echo "Usage: $0 <repository_name>"
    exit 1
fi

REPO_NAME="$1"
UNTIL="2024-01-12T10:30:00Z"
URL="http://localhost:3000/v1/repositories/${REPO_NAME}/commits?until=${UNTIL}"

# Make the HTTP GET request silently,
# capturing only the HTTP status code.
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$URL")

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo "success"
else
    echo "Error: HTTP status $HTTP_STATUS"
fi