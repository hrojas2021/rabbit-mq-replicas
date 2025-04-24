#!/bin/bash

URL="http://localhost:8080/publish"

for i in {1..20}
do
  MESSAGE="This is a HTTP Message $i"
  echo "Sending: $MESSAGE"
  curl -s -X POST $URL -H "Content-Type: application/json" -d "{\"body\": \"$MESSAGE\"}"
  echo
  sleep 1
done
