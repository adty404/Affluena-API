.PHONY: verify seed

verify:
	./scripts/verify.sh

seed:
	go build -o /tmp/affluena-seed ./cmd/seed
	/tmp/affluena-seed
	@rm -f /tmp/affluena-seed
