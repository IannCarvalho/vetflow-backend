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
| `AdminResetToken` | `ADMIN_RESET_TOKEN` | — (vazio = `/admin/reset-demo` desabilitado) |
| `RegisterInviteCode` | `REGISTER_INVITE_CODE` | — (produção; vazio = cadastro clássico fechado) |
| `MaxProductionUsers` | `MAX_PRODUCTION_USERS` | `0` (sem teto) |

> `DEMO_RESET_INTERVAL_HOURS` e `SEED_FORCE` **não** vivem na struct `Config` —
> são lidos diretamente em [`../db/seed.go`](../db/README.md).

A lista canônica de variáveis vive em [`../.env.example`](../.env.example).

## Comportamentos importantes

- **`Validate()` falha fechado**: sem `JWT_SECRET` a aplicação se recusa a
  iniciar em **qualquer** ambiente — nunca há segredo padrão (impede forjar
  tokens para qualquer usuário).
- **`IsProduction()`**: `APP_ENV` igual a `production`/`prod` (case-insensitive).
  Usado para pular seed e desabilitar comportamentos de dev.
- **`IsDemo()`**: `APP_ENV` igual a `demo` (case-insensitive). Modo demo roda o
  mesmo código de produção, porém contra um banco descartável com dados falsos.
- **Guarda de isolamento**: `Validate()` recusa iniciar quando `APP_ENV` é
  não-produção **e** `MONGO_DB` aponta para o banco de produção (`vetflow`),
  evitando que o seed/visitor login corrompa dados reais.
- **Mascaramento**: `maskPassword` (usado no log de boot e em `String()`)
  troca a senha do userinfo do `MONGO_URI` por `****`
  (`mongodb+srv://app:****@cluster/...`), preservando host, banco e opções.
  URIs sem senha saem inalteradas.

## Quem consome

`cmd/main.go` chama `LoadConfig()` e `Validate()` no bootstrap; os valores são
passados a `db`, `routes` e `services`.
