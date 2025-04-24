#!/bin/bash
# wait-for-it.sh

# The script waits for a given host and port to be available

HOST=$1
PORT=$2

while ! nc -z $HOST $PORT; do
  echo "Waiting for $HOST:$PORT..."
  sleep 2
done

echo "$HOST:$PORT is available!"
exec "$@"
