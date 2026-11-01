# Tools for playground-openapi.yaml maintenance.
#
# Required tools:
# - docker or podman
# - mise (see ../.config/mise.toml)   https://mise.jdx.dev/
#
# All other tools are installed via mise.

# $(call which,prog): first path of prog (or prog.exe) found in $PATH, or empty.
# Pure GNU make (no shell): works with both ':' (Unix) and ';' (Windows) PATH
# separators. Spaces in PATH entries are replaced by the '?' wildcard so that
# directories such as "C:\Program Files\..." are matched.
empty :=
space := $(empty) $(empty)
pathsep := $(if $(findstring ;,$(PATH)),;,:)
path_dirs := $(subst $(pathsep), ,$(subst \,/,$(subst $(space),?,$(PATH))))
which = $(firstword $(wildcard $(foreach d,$(path_dirs),$(d)/$(1) $(d)/$(1).exe)))

# Use docker if available in PATH, else podman.
# Override with: make docker=podman
docker ?= $(if $(call which,docker),docker,podman)

.PHONY: openapi-validate

openapi-validate: playground-openapi.yaml .spectral.yml
	$(docker) run --rm -v "$(CURDIR):/src" -e NODE_NO_WARNINGS=1 docker.io/stoplight/spectral lint -r /src/.spectral.yml /src/playground-openapi.yaml

.PHONY: swagger-ui

PORT ?= 8080

swagger-ui:
	( sleep 2 ; open http://localhost:$(PORT)/ ) &
	$(docker) run --rm -p 127.0.0.1:$(PORT):8080 -v "$(CURDIR):/src" -e SWAGGER_JSON=/src/playground-openapi.yaml docker.io/swaggerapi/swagger-ui
