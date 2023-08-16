#!/bin/bash
printf '%s' "$(git describe --tags $(git rev-list --tags --max-count=1))" > version.txt
