.PHONY: generate-migrations migrate-gen help

# Migration generator tool
MIGRATE_GEN := go run ./cmd/migrate-gen

help:
	@echo "Available targets:"
	@echo "  generate-migrations  - Generate SQL migrations from Go models"
	@echo "  migrate-gen-dry      - Preview migrations without writing files"
	@echo "  help                 - Show this help message"

# Generate migrations from models
generate-migrations:
	@echo "Generating migrations from models..."
	$(MIGRATE_GEN) -output migrations/

# Dry run - preview migrations without writing
migrate-gen-dry:
	@echo "Previewing migrations (dry run)..."
	$(MIGRATE_GEN) -output migrations/ -dry-run -verbose

# Generate migrations for specific models
migrate-gen-models:
	@echo "Usage: make migrate-gen-models MODELS='Model1,Model2'"
	@if [ -z "$(MODELS)" ]; then \
		echo "Error: MODELS variable is required"; \
		exit 1; \
	fi
	$(MIGRATE_GEN) -output migrations/ -models-list "$(MODELS)"
