.PHONY: clean
clean:
	find . -type d -name fakes -exec rm -rf {} \;  || true

.PHONY: generate
generate:
	go generate ./...
	gsed -i '/new(FakeVarzReporter)/i \/\/lint:ignore SA1019 - auto-generated fake will go away when Varz goes away' gorouter/metrics/fakes/fake_varzreporter.go
