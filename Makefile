.ONESHELL:
all:
	git submodule update --init --recursive
	cd site/
	yarn --non-interactive
	NODE_ENV=production `yarn bin`/webpack
	statik -src=./dist
	cd ..
	go build
