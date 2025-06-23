

.PHONY: all

all: go-logo-blue.png

%.png: %.svg
	magick -background none $< -colors 2 $@

go-logo-blue.svg:
	curl -O https://go.dev/images/go-logo-blue.svg
