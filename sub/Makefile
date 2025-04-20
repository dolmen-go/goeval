
.PHONY: openapi-validate

openapi-validate: playground-openapi.yaml .spectral.yml
	docker run --rm -it -v "$(CURDIR):/src" stoplight/spectral lint -r /src/.spectral.yml /src/playground-openapi.yaml

.PHONY: swagger-ui

PORT ?= 8080

swagger-ui:
	( sleep 2 ; open http://localhost:$(PORT)/ ) &
	docker run --rm -p 127.0.0.1:$(PORT):8080 -v "$(CURDIR):/src" -e SWAGGER_JSON=/src/playground-openapi.yaml swaggerapi/swagger-ui
