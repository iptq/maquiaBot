all: maquiaBot

maquiaBot:
	go build -v

test:
	go test -v ./...

checkFmt:
	[ -z "$$(git ls-files | grep '\.go$$' | xargs gofmt -l)" ] || (exit 1)

watch:
	watchexec "bash -c 'go build |& tee >(wc -l)'"

clean:
	rm -f maquiaBot

.PHONY: all clean test checkFmt watch

