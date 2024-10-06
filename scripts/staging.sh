#!/bin/bash

export SLUG=ghcr.io/awakari/int-email
export VERSION=latest
docker tag awakari/int-email "${SLUG}":"${VERSION}"
docker push "${SLUG}":"${VERSION}"
