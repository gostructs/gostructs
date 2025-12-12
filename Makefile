CYAN := \033[36m
GREEN := \033[32m
RED := \033[31m
NC := \033[0m

check:
	@$(MAKE) ver-checks || $(MAKE) install-checks
	@printf "$(CYAN)*** goimports…$(NC)\n"
	@goimports -w .
	@printf "$(CYAN)*** deadcode…$(NC)\n"
	@deadcode ./...
	@printf "$(CYAN)*** gostructs…$(NC)\n"
	@go run . ./...
	@printf "$(CYAN)*** go fmt…$(NC)\n"
	@go fmt ./...
	@printf "$(CYAN)*** go mod tidy…$(NC)\n"
	@go mod tidy
	@printf "$(CYAN)*** go vet…$(NC)\n"
	@go vet ./...
	@printf "$(CYAN)*** staticcheck…$(NC)\n"
	@staticcheck ./...
	@printf "$(CYAN)*** golangci-lint…$(NC)\n"
	@golangci-lint run --enable=unused ./...
	@printf "$(CYAN)*** govulncheck…$(NC)\n"
	@govulncheck ./...
	@printf "$(CYAN)*** gosec…$(NC)\n"
	@gosec -severity=HIGH -confidence=HIGH ./...
	@printf "$(GREEN)*** Lint & security checks passed!$(NC)\n"

install-checks:
	@printf "$(CYAN)*** Installing check tools…$(NC)\n"
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/deadcode@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	@printf "$(GREEN)*** All check tools installed!$(NC)\n"

ver-checks:
	@printf "$(CYAN)*** Verifying check tools…$(NC)\n"
	@command -v goimports >/dev/null 2>&1 || { printf "$(RED)goimports not found$(NC)\n"; exit 1; }
	@command -v deadcode >/dev/null 2>&1 || { printf "$(RED)deadcode not found$(NC)\n"; exit 1; }
	@command -v staticcheck >/dev/null 2>&1 || { printf "$(RED)staticcheck not found$(NC)\n"; exit 1; }
	@command -v golangci-lint >/dev/null 2>&1 || { printf "$(RED)golangci-lint not found$(NC)\n"; exit 1; }
	@command -v govulncheck >/dev/null 2>&1 || { printf "$(RED)govulncheck not found$(NC)\n"; exit 1; }
	@command -v gosec >/dev/null 2>&1 || { printf "$(RED)gosec not found$(NC)\n"; exit 1; }
	@printf "$(GREEN)*** All check tools available!$(NC)\n"

test:
	@printf "$(CYAN)*** Running tests…$(NC)\n"
	@go test -v ./...

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.1")
NEXT_VERSION ?= $(VERSION)

release:
	@if [ -z "$(v)" ]; then \
		printf "$(RED)Usage: make release v=X.Y.Z$(NC)\n"; \
		exit 1; \
	fi
	@printf "$(CYAN)*** Creating release v$(v)…$(NC)\n"
	@git diff --quiet || { printf "$(RED)Uncommitted changes. Commit first.$(NC)\n"; exit 1; }
	@git tag -a v$(v) -m "Release v$(v)"
	@git push origin v$(v)
	@printf "$(GREEN)*** Tag v$(v) pushed! Creating GitHub release…$(NC)\n"
	@gh release create v$(v) --title "v$(v)" --generate-notes
	@printf "$(GREEN)*** Release v$(v) created!$(NC)\n"
	@printf "$(CYAN)*** SHA256:$(NC)\n"
	@curl -sL https://github.com/afify/gostructs/archive/refs/tags/v$(v).tar.gz | shasum -a 256

.PHONY: check install-checks ver-checks test release
