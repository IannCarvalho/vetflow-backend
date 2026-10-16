# `db/` — Conexão, índices e seed do MongoDB

[⬆️ Voltar para o README do backend](../README.md)

Camada de infraestrutura de persistência. Abre e fecha a conexão com o MongoDB,
garante os índices usados pelas consultas e popula dados iniciais fora de
produção.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `mongo.go` | `NewMongoConnection` / `CloseConnection` — ciclo de vida da conexão. |
| `indexes.go` | `CreateIndexes` — índices idempotentes por coleção. |
| `seed.go` | `SeedDatabase` / `ResetDemoDatabase` / `MaybeResetDemoOnBoot` — dados de teste/demo (nunca em produção) e reset da demo. |

## Conexão (`mongo.go`)

- `NewMongoConnection(mongoURI)` aplica a URI, define
  `ServerSelectionTimeout` (30s) e `ConnectTimeout` (15s), e valida com um
  `Ping` (15s). Em falha de ping faz `Disconnect` best-effort.
- `CloseConnection(client)` desconecta com timeout de 10s (chamado via `defer`
  no `cmd/main.go`).

## Índices (`indexes.go`)

`CreateIndexes` é **idempotente** — seguro rodar a cada boot. Cria:

- **Single-field** por coleção (ex.: `clients.vet_id`, `pets.owner_id`,
  `attachments.pet_id`).
- **Compound** para as contagens do dashboard (ex.: `{vet_id, date}` em
  consultations, `{vet_id, status}` em invoices, `{vet_id, start_time}` em
  appointments).
- **Composite ordenados** para listagens por data (ex.:
  `{vet_id:1, issue_date:-1}` em invoices).

> ⚠️ `pets` é indexado por `owner_id`, **não** `vet_id` — pets são escopados
> pelo `vet_id` do cliente dono. Os nomes de campo batem exatamente com as tags
> BSON dos [`models`](../models/README.md) e com os filtros dos
> [`repositories`](../repositories/README.md).

## Seed (`seed.go`)

`SeedDatabase` insere contas-demo e dados de exemplo. **Nunca roda em
produção** — `cmd/main.go` pula o seed quando `cfg.IsProduction()` é verdadeiro,
pois a senha das contas é conhecida (seria um backdoor de credenciais).
`SEED_FORCE=true` força um wipe + re-seed limpo no boot (fora de produção).

- `ResetDemoDatabase(db)` limpa e re-semeia o banco da demo (também protegido
  contra rodar em produção). É o que o endpoint `POST /admin/reset-demo` chama.
- `MaybeResetDemoOnBoot(db)` roda **apenas em modo demo**: re-semeia no boot se o
  último seed for mais antigo que `DEMO_RESET_INTERVAL_HOURS` (default 6; `0` =
  todo boot). Mantém a demo auto-limpa sem agendador externo.
