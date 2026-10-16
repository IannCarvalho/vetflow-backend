# `routes/` — Roteamento, middleware e rate limiting

[⬆️ Voltar para o README do backend](../README.md)

Monta o roteador HTTP (Chi v5), faz a injeção de dependências (repositórios →
serviços → controllers) e aplica os middlewares transversais: CORS, logging,
recover, autenticação JWT e rate limiting das rotas de credenciais.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `router.go` | `NewRouter(db)` — monta todas as rotas e injeta dependências. |
| `middleware.go` | `AuthMiddleware` — valida JWT e injeta o `UserID` no contexto. |
| `rate_limiter.go` | `NewRateLimiter` — limite por IP em janela fixa de 1 min. |
| `rate_limiter_test.go` | Testes do rate limiter. |

## `router.go`

É o **composition root** do backend:

1. Configura **CORS** a partir de `CORS_ALLOWED_ORIGINS` (lista separada por
   vírgula; vazio = `*`, comportamento de dev).
2. Aplica `middleware.Logger` e `middleware.Recoverer`.
3. Instancia **todos** os repositórios, serviços e controllers (DI explícita),
   incluindo a `RegistrationPolicy` (invite code + teto de contas; ativa apenas
   em produção, lida de `REGISTER_INVITE_CODE` / `MAX_PRODUCTION_USERS`).
4. Expõe `GET /health` (readiness, com ping ao Mongo; não autenticado).
5. Expõe `POST /admin/reset-demo` (não-produção): protegido por três camadas —
   `APP_ENV` não-produção (senão `404`), `X-Admin-Token` igual a
   `ADMIN_RESET_TOKEN` (comparação em tempo constante) e a guarda interna de
   `db.ResetDemoDatabase`. Limpa e re-semeia o banco da demo.
6. Serve a documentação Swagger em `/swagger/*`.
7. Registra os grupos de rotas: `/auth`, `/users`, `/clients`, `/pets`,
   `/consultations`, `/vaccines`, `/prescriptions`, `/catalog`, `/invoices`,
   `/appointments`, `/attachments`, `/dashboard`, `/notifications`.

Todas as rotas de recurso usam `AuthMiddleware` e são escopadas ao vet
autenticado.

## `middleware.go` — `AuthMiddleware`

Extrai o token de `Authorization: Bearer <token>`, valida com
`helpers.ParseToken` (ver [`../helpers/README.md`](../helpers/README.md)) e
injeta `ctxkeys.UserID` no contexto da request. Falhas retornam `401` em JSON.

## `rate_limiter.go` — `NewRateLimiter`

Middleware por IP com janela fixa de 1 minuto. `AUTH_RATE_LIMIT_PER_MIN` define
o limite (default 10; `<= 0` desabilita). Resolve o IP via `X-Forwarded-For`
(primeiro valor) ou `RemoteAddr`. Aplicado às rotas de credencial em `/auth`
(register, login, social-login, oauth/callback, verify) para blindar
brute-force e replay de token. Excesso retorna `429`.
