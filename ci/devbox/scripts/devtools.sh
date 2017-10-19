#!/bin/bash -e

echo "Install development tools."
yum groupinstall -y "Development Tools"
yum install -y rpm-build rpmdevtools

echo "Install golang."
# Requires to export PATH=/usr/local/go/bin:$PATH
curl -L https://storage.googleapis.com/golang/go1.9.linux-amd64.tar.gz | tar -C /usr/local -xzf -

