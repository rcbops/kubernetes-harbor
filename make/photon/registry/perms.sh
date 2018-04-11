#!/bin/sh

set -e

if [ -d /etc/registry ]; then
    chown 10000:10000 -R /etc/registry
fi
if [ -d /var/lib/registry ]; then
    chown 10000:10000 -R /var/lib/registry
fi  
if [ -d /storage ]; then
    chown 10000:10000 -R /storage
fi