# `cmd/` — Application entry point

[⬆️ Voltar para o README do backend](../README.md)

Este diretório contém o ponto de entrada executável do VetFlow backend. É aqui
que o processo nasce, carrega a configuração, abre a conexão com o banco e sobe
o servidor HTTP.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `main.go` | `func main()`: bootstrap e ciclo de vida completo do servidor. |

## O que o `main.go` faz, em ordem

1. **Contexto de shutdown** — cria um `context` cancelado em `SIGINT`/`SIGTERM`
   para propagar o encerramento gracioso.
2. **Carrega a configuração** via `config.LoadConfig()` e falha rápido em
   `cfg.Validate()` (ex.: `JWT_SECRET` ausente derruba o boot).
3. **Conecta ao MongoDB** com `db.NewMongoConnection(cfg.MongoURI)` e registra o
   `CloseConnection` no `defer`.
4. **Seed** — roda `db.SeedDatabase` apenas fora de produção
   (`cfg.IsProduction()` evita injetar contas-demo com senha conhecida em prod).
5. **Índices** — `db.CreateIndexes` com timeout de 30s; falhas apenas logam e
   nunca bloqueiam o boot.
6. **Router** — monta o roteador Chi com `routes.NewRouter(database)`.
7. **Scheduler de notificações** — se `EMAIL_NOTIFICATIONS_ENABLED`, inicia o
   `services.NotificationScheduler` (envio diário de e-mails, não bloqueante).
8. **Servidor HTTP** — `http.Server` com timeouts de leitura/escrita/idle
   (mitigação de slowloris) escutando na porta de `cfg.Port`.
9. **Shutdown gracioso** — ao receber o sinal, chama `srv.Shutdown` com timeout
   de 10s.

As anotações Swagger (`// @title`, `// @version`, `// @BasePath`, ...) no topo
do arquivo são a fonte da documentação OpenAPI gerada em [`../docs`](../docs).

## Como executar

Veja o [README do backend](../README.md) para build/run via Docker Compose e
variáveis de ambiente. Resumo:

```bash
go run ./cmd        # a partir da raiz do vetflow-backend
```

## Dependências diretas

`config`, `db`, `routes`, `repositories`, `services`, `helpers`.
