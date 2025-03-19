ALL_PACKAGES:=./...
CMD_PACKAGES:=./cmd/main.go

.PHONY: init fmt lint lint-fix build run test cover clean

all: fmt lint test cover build

# プロジェクトの初期セットアップ
init:
	go mod tidy

# コードフォーマット
fmt:
	go fmt ${ALL_PACKAGES}

# Lint チェック
lint:
	golangci-lint run ${ALL_PACKAGES}

# Lint の修正適用
lint-fix:
	golangci-lint run ${ALL_PACKAGES} --fix

# ビルド
build:
	go build -o bin/server ${CMD_PACKAGES}

# サーバー実行
run:
	go run ${CMD_PACKAGES}

# テスト実行
test:
	go test -v ${ALL_PACKAGES}

# カバレッジ付きテスト実行
cover:
	mkdir -p coverage
	go test -cover ${ALL_PACKAGES} -coverprofile=coverage/cover.out
	go tool cover -html=coverage/cover.out -o coverage/cover.html

# クリーンアップ
clean:
	rm -rf bin/ coverage/