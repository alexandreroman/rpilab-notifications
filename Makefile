IMAGE ?= ghcr.io/alexandreroman/rpilab-notifications

all: build

build:
	docker build -t $(IMAGE) .

clean:
