.PHONY: install

install:
	go build -o out/mf . && sudo cp out/mf /usr/local/bin/mf
