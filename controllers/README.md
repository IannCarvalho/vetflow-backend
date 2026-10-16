# `controllers/` — Handlers HTTP (camada C do MVC)

[⬆️ Voltar para o README do backend](../README.md)

Cada controller recebe a requisição HTTP, valida a entrada, aplica a regra de
autorização (ownership por `vet_id`), delega ao repositório/serviço e escreve a
resposta JSON. É a fronteira entre o mundo HTTP e o domínio.

## Controllers

| Arquivo | Recurso / rota |
| --- | --- |
| `auth_controller.go` | `/auth` — register, login, verify. |
| `social_auth_controller.go` | `/auth/social-login` — valida ID token do Google server-side. |
| `oauth_callback_controller.go` | `/auth/oauth/callback` — emite JWT real. |
| `registration_policy.go` | Política de criação de conta em produção (invite code + teto de usuários); compartilhada por auth/social/OAuth. |
| `user_controller.go` | `/users` — perfil, foto, troca de senha. |
| `client_controller.go` | `/clients`. |
| `pet_controller.go` | `/pets`. |
| `consultation_controller.go` | `/consultations` (+ draft de receita). |
| `vaccine_controller.go` | `/vaccines`. |
| `prescription_controller.go` + `prescription_print.go` | `/prescriptions` (+ `/print`). |
| `catalog_item_controller.go` | `/catalog`. |
| `invoice_controller.go` | `/invoices`. |
| `appointment_controller.go` | `/appointments`. |
| `attachment_controller.go` | `/attachments`. |
| `dashboard_controller.go` | `/dashboard/stats`. |
| `notification_controller.go` | `/notifications`. |
| `helpers.go` | Utilitários compartilhados (abaixo). |
| `*_test.go`, `fakes_test.go` | Testes + fakes dos repositórios. |

## `helpers.go` — utilitários compartilhados

- `writeError` / `writeErrorWithDetails` — respostas de erro JSON padronizadas.
- `getUserIDFromContext(ctx)` — lê o `UserID` injetado pela `AuthMiddleware`
  (via `internal/ctxkeys`) e converte para `ObjectID`.
- `validateResourceOwnership` / `verifyClientOwnership` — garantem que o recurso
  pertence ao vet autenticado; sentinelas `errClientNotFound` /
  `errClientAccessDenied` deixam o mapeamento de status (400/403/404/500) a
  cargo de cada handler.

## Convenções

- **Multi-tenant**: todo handler protegido resolve o `userID` do contexto e
  valida ownership antes de ler/escrever.
- Dependências (repositórios/serviços) são injetadas via construtores
  `New<X>Controller(...)` chamados em
  [`../routes/README.md`](../routes/README.md).
- Validação de entrada usa os DTOs e `Validate()` dos
  [`../models/README.md`](../models/README.md).
