#!/bin/bash

# First time deployment; config is copied to host.
# Config directory mounted as volume for app to utilize external config.

# Stop potentially running container:
docker compose down

# Copy config file into /home/user:
./config_copy.sh

# Start container using
docker compose up -d