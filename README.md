# logs-collector-api

## 概要

`logs-collector-api` は、分散システム向けのログ収集・検索 API サーバーです。
REST および gRPC インターフェースを提供し、各種サービスからのログを受信・保存・検索できます。
バックエンドには PostgreSQL、Elasticsearch、NATS を利用しています。

## 主な機能

- **ログの登録**
  サービス名・レベル・メッセージ・トレース ID・メタデータ等を含むログを API 経由で登録できます。
- **ログの検索**
  サービス名・レベル・ページネーション等の条件でログを検索できます。
- **REST/gRPC 両対応**
  REST API と gRPC の両方で操作可能です。
- **分散トレーシング対応**
  traceID によるログの追跡が可能です。

## API 仕様

### REST エンドポイント

- `GET /logs`
  クエリパラメータ（service, level, limit, offset）でログを検索
- `POST /logs`
  ログを新規登録

#### ログデータ例

```json
{
  "id": "UUID",
  "level": "info",
  "message": "ログメッセージ",
  "service": "frontend",
  "timestamp": "2024-01-01T00:00:00Z",
  "traceId": "xxxx-xxxx",
  "metadata": {
    "ip": "127.0.0.1"
  }
}
```

詳細な API 仕様は `docs/swagger.yaml` または `docs/swagger.json` を参照してください。

### gRPC エンドポイント

- サービス定義は [logs-collector-protos](https://github.com/KeitaShimura/logs-collector-protos) の `logs/v1/logs.proto` を参照してください。
- 主なメソッド:
  - `SendLog(SendLogRequest)`: ログを新規登録
  - `GetLogs(GetLogsRequest)`: クエリ条件でログを取得
- デフォルトポート: `50051`

## 必要な環境変数

| 変数名               | 必須 | デフォルト値          | 説明                        |
| -------------------- | ---- | --------------------- | --------------------------- |
| DB_HOST              | 必須 |                       | PostgreSQL ホスト           |
| DB_PORT              | 必須 |                       | PostgreSQL ポート           |
| DB_NAME              | 必須 |                       | データベース名              |
| DB_USER              | 必須 |                       | データベースユーザー        |
| DB_PASS              | 必須 |                       | データベースパスワード      |
| DB_SSLMODE           |      | disable               | SSL モード                  |
| DB_TIMEZONE          |      | Asia/Tokyo            | タイムゾーン                |
| NATS_URL             |      | nats://nats:4222      | NATS サーバー URL           |
| ELASTICSEARCH_URL    |      | http://localhost:9200 | Elasticsearch サーバー URL  |
| GRPC_PORT            |      | 50051                 | gRPC サーバーポート         |
| GRPC_BIND_ADDR       |      | 0.0.0.0               | gRPC バインドアドレス       |
| REST_PORT            |      | 8080                  | REST サーバーポート         |
| REST_BIND_ADDR       |      | 0.0.0.0               | REST バインドアドレス       |
| GRPC_TIMEOUT_SEC     |      | 2                     | gRPC リクエストタイムアウト |
| REST_TIMEOUT_SEC     |      | 3                     | REST リクエストタイムアウト |
| SHUTDOWN_TIMEOUT_SEC |      | 10                    | シャットダウン猶予秒数      |
| LOG_LEVEL            |      | INFO                  | ログレベル                  |

## セットアップ

### 1. DB マイグレーション

```sh
make migrate
```

### 2. サーバー起動

#### ローカル開発

```sh
make run
```

#### Docker 利用

```sh
docker-compose up
```

## テスト

```sh
make test
```

## 依存技術

- Go 1.24.x
- PostgreSQL
- Elasticsearch
- NATS
- Echo (REST)
- gRPC
