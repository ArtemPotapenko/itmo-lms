GO_TEST=go test

.PHONY: test test-unit test-integration test-e2e

test: test-unit test-integration test-e2e

test-unit:
	$(GO_TEST) ./auth-service/... ./content-service/internal/application ./course-service/... ./document-service/... ./statistic-service/internal/application ./pkg/...

test-integration:
	$(GO_TEST) ./content-service/internal/infrastructure/postgres ./statistic-service/internal/infrastructure/postgres ./document-service/internal/infrastructure/postgres

test-e2e:
	$(GO_TEST) ./content-service/e2e ./document-service/e2e
