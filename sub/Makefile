
.PHONY: openapi-validate

openapi-validate: playground-openapi.yaml .spectral.yml
	docker run --rm -it -v "$(CURDIR):/src" stoplight/spectral lint -r /src/.spectral.yml /src/playground-openapi.yaml
