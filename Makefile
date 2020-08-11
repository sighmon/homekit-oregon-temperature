GOCMD=go
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
GORUN=$(GOCMD) run

export GO111MODULE=on

build:
	$(GOGET)
	$(GOBUILD) homekit-oregon-scientific-idtw211r.go

run:
	$(GORUN) homekit-oregon-scientific-idtw211r.go
