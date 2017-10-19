#!/bin/bash -e

env
go version

echo "Build rpm leeroy."
rpmdev-setuptree
rpmbuild -bb ./pkg/rhel/leeroy.spec
cp -r $HOME/rpmbuild/RPMS /mnt/artifact

