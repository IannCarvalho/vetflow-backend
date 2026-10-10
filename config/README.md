# `config/` — Configuração da aplicação

[⬆️ Voltar para o README do backend](../README.md)

Carrega e valida toda a configuração do backend a partir de variáveis de
ambiente (com suporte a `.env` via `godotenv`). É a única fonte de verdade das
configurações em runtime.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `config.go` | Struct `Config`, `LoadConfig`, `Validate`, helpers de env. |
| `config_test.go` | Testes de carregamento e validação. |

## A struct `Config`

| Campo | Variável de ambiente | Default |
| --- | --- | --- |
| `MongoURI` | `MONGO_URI` | `mongodb://mongo:27017` |
| `MongoDB` | `MONGO_DB` | `vetflow` |
| `Port` | `PORT` | `8080` |
| `AppEnv` | `APP_ENV` | `development` |
| `JWTSecret` | `JWT_SECRET` | — (obrigatório) |
| `SMTPHost` | `SMTP_HOST` | vazio = modo "log only" |
| `SMTPPort` | `SMTP_PORT` | `587` |
| `SMTPUser` / `SMTPPass` | `SMTP_USER` / `SMTP_PASS` | — |
| `SMTPFrom` | `SMTP_FROM` | — |
| `FromName` | `SMTP_FROM_NAME` | `VetFlow` |
| `EmailNotificationsEnabled` | `EMAIL_NOTIFICATIONS_ENABLED` | `true` |

A lista canônica de variáveis vive em [`../.env.example`](../.env.example).

## Comportamentos importantes

- **`Validate()` falha fechado**: sem `JWT_SECRET` a aplicação se recusa a
  iniciar em **qualquer** ambiente — nunca há segredo padrão (impede forjar
  tokens para qualquer usuário).
- **`IsProduction()`**: `APP_ENV` igual a `production`/`prod` (case-insensitive).
  Usado para pular seed e desabilitar comportamentos de dev.
- **Mascaramento**: `maskPassword` e `String()` evitam vazar credenciais do
  `MONGO_URI` nos logs.

## Quem consome

`cmd/main.go` chama `LoadConfig()` e `Validate()` no bootstrap; os valores são
passados a `db`, `routes` e `services`.
