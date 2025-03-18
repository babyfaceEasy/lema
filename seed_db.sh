#!/bin/bash

read -p "Enter repository name: " REPO_NAME
read -p "Enter owner name: " OWNER_NAME
read -p "Enter start date (format: 2025-03-12T10:35:55Z) [optional]: " START_DATE

if [ -z "$START_DATE" ]; then
    DATA="{\"repo_name\": \"$REPO_NAME\", \"owner_name\": \"$OWNER_NAME\"}"
else
    DATA="{\"repo_name\": \"$REPO_NAME\", \"owner_name\": \"$OWNER_NAME\", \"start_time\": \"$START_DATE\"}"
fi

# URL for the POST call.
URL="http://localhost:3000/v1/repositories/monitor"

echo "Sending POST request to $URL with payload:"
echo "$DATA"

HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Content-Type: application/json" -d "$DATA" "$URL")

if [ "$HTTP_STATUS" -eq 200 ]; then
    echo "success"
else
    echo "Error: HTTP status $HTTP_STATUS"
fi
