ALL_PACKAGES := ./...                                       # 全てのGoパッケージ
CMD_PACKAGES := ./cmd/main.go                               # メイン実行ファイル
SQLBOILER_CONFIG := internal/infra/db/config/sqlboiler.toml # SQLBoilerの設定ファイル
SQLBOILER_OUTPUT := internal/infra/db/models                # SQLBoilerの出力先
SWAG_MAIN := cmd/main.go                                    # swag init でのエントリーポイント
SWAG_OUT := docs                                            # Swagger ドキュメントの出力先ディレクトリ

.PHONY: init format lint lint-fix build run test cover generate clean swagger migrate all

# すべての主要なタスクを順に実行
all: format lint test cover build

# プロジェクトの初期セットアップ（依存関係の整備）
init:
	go mod tidy

# コードフォーマットとインポート整理
format:
	go fmt ${ALL_PACKAGES}
	gci write -s standard -s default -s "prefix(github.com/KeitaShimura)" $(shell find . -name '*.go')
	gofumpt -w .

# Lint チェック（静的解析）
lint:
	golangci-lint run

# Lint の自動修正
lint-fix:
	golangci-lint run --fix

# バイナリのビルド
build:
	go build -o bin/server ${CMD_PACKAGES}

# サーバーの実行（開発用）
run:
	go run ${CMD_PACKAGES}

# テスト実行（詳細出力付き）
test:
	go test -v ${ALL_PACKAGES}

# カバレッジ付きテストの実行とHTMLレポート出力
cover:
	mkdir -p coverage
	go test -cover ${ALL_PACKAGES} -coverprofile=coverage/cover.out
	go tool cover -html=coverage/cover.out -o coverage/cover.html

# コード生成（SQLBoilerによるDBモデル生成）
generate:
	rm -rf ${SQLBOILER_OUTPUT}
	sqlboiler psql --config ${SQLBOILER_CONFIG} --output ${SQLBOILER_OUTPUT} --no-tests

# 生成物と一時ファイルのクリーンアップ
clean:
	rm -rf bin/ coverage/ ${SQLBOILER_OUTPUT}

# Swagger ドキュメント生成
swagger:
	swag init --generalInfo ${SWAG_MAIN} --output ${SWAG_OUT}

# データベースマイグレーションの実行
migrate:
	go run internal/infra/db/migrations/migrate.go
