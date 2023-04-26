#!/usr/bin/env bash
CGO_ENABLED=0 go build -o bbb-stress-test
echo "Done! Now just run ./bbb-stress-test"
