#!/bin/bash

SRCS=$(find . -name '*.go')
echo $SRCS
for file in $SRCS; do
    echo $file
		golint ${file};
done
