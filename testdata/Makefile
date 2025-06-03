

.PHONY: all

all: go-logo-blue.png

go-logo-blue.png: go-logo-blue.svg
	magick -background none $< -colors 2 $@

go-logo-blue.svg:
	curl -O https://go.dev/images/go-logo-blue.svg
