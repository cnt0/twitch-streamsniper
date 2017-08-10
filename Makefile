.ONESHELL:
all:
	git submodule update --init --recursive
	cd site/
	yarn --non-interactive
	ng build -prod -sm false
	statik -src=./dist
	cd ..
	go build
