#!/bin/bash
set -e

HOST="cassandra"
DEFAULT_PASSWORD="cassandra"
TARGET_PASSWORD="${CASSANDRA_PASSWORD}"

if [ -z "${TARGET_PASSWORD}" ]; then
  echo "CASSANDRA_PASSWORD is not set, skipping"
  exit 0
fi

if [ "${TARGET_PASSWORD}" = "${DEFAULT_PASSWORD}" ]; then
  echo "CASSANDRA_PASSWORD matches default, nothing to change"
  exit 0
fi

echo "Waiting for Cassandra to accept CQL connections..."
until cqlsh "${HOST}" -u cassandra -p "${DEFAULT_PASSWORD}" -e "DESCRIBE CLUSTER" >/dev/null 2>&1; do
  if cqlsh "${HOST}" -u cassandra -p "${TARGET_PASSWORD}" -e "DESCRIBE CLUSTER" >/dev/null 2>&1; then
    echo "Password already updated, skipping"
    exit 0
  fi
  sleep 2
done

cqlsh "${HOST}" -u cassandra -p "${DEFAULT_PASSWORD}" \
  -e "ALTER ROLE cassandra WITH PASSWORD = '${TARGET_PASSWORD}'"

echo "Cassandra password updated successfully"
