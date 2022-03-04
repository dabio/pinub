#!/bin/sh

# Create a new migration file for the database

if [[ -z "$1" ]]; then
    echo "Please provide a name for this migration."
    exit 1
fi

command -v migrate >/dev/null 2>&1 || {
    echo >&2 "Migrate command not found. Have you installed golang-migrate?";
    echo >&2 "https://github.com/golang-migrate/migrate/blob/master/cmd/migrate/README.md#installation";
    exit 1;
}

migrate create -ext sql -dir migrations -seq $1

# Template for the newly created migration file
TEMPLATE='BEGIN;
-- Author the migration here.
END;'

for m in $(ls migrations | tail -n 2); do echo "$TEMPLATE" >> "migrations/$m"; done
