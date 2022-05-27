GOVERSION=1.16.11
GOIMG=golang:$(GOVERSION)-stretch

VOLS=-v `pwd`:/app -w /app
RUNNET=--net=host
RUNENVS=-e "FIRESTORE_EMULATOR_HOST=127.0.0.1:8080"
RUNGO=docker run -it --rm $(VOLS) $(RUNNET) $(RUNENVS) $(GOIMG)

FSCONTAINER=firestore-emulator
FSPORT=8080
RUNFSENVS=-e "FIRESTORE_PROJECT_ID=test" -e "PORT=$(FSPORT)"
RUNFSNET=-p $(FSPORT):$(FSPORT)
FSIMG=mtlynch/firestore-emulator-docker
RUNFS=docker run -d --rm --name $(FSCONTAINER) $(RUNFSENVS) $(RUNFSNET) $(FSIMG)

deps:
	$(RUNGO) go mod tidy
	$(RUNGO) go mod vendor

test:
	-docker stop $(FSCONTAINER)
	$(RUNFS)
	$(RUNGO) go test -race -cover ./...
	-docker stop $(FSCONTAINER)

.PHONY: deps test