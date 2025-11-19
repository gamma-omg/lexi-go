.PHONY: init
init:
	sh scripts/install-tools.sh

.PHONY: proto
proto:
	sh scripts/proto.sh api