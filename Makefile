.PHONY:all

all:Deepl-translator-linux

Deepl-translator-linux:
	$(shell go build -o Deepl-translator-linux)

clean:
	@-rm Deepl-translator-linux
