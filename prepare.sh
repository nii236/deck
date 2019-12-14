#!/bin/bash
set -e
rm -rf deploy
mkdir deploy
go build -o deploy/deck
cp -r ~/japanese/output ./deploy
mkdir deploy/templates
cp -r ~/git/deck/view.gotpl deploy/templates/  
cp -r ~/git/deck/init deploy/