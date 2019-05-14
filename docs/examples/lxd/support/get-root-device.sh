#!/bin/sh

ROOT_DEV=$(mount | grep " on / " | cut -f1 -d" ")

echo '{"device":"'$ROOT_DEV'"}'