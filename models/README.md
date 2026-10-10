# `models/` — Entidades de domínio e DTOs

[⬆️ Voltar para o README do backend](../README.md)

Define as structs de domínio persistidas no MongoDB (com tags `bson`/`json`),
os DTOs de request/response e as validações de entrada. Nenhuma dependência de
banco ou HTTP — apenas tipos e regras de validação.

## Entidades

| Arquivo | Entidade | Observações |
| --- | --- | --- |
| `user.go` | `User` + `RegisterRequest`, `LoginRequest`, `AuthResponse`, `SocialLoginRequest` | Senha nunca serializada (`json:"-"`). |
| `client.go` | `Client` | Escopado por `vet_id`. |
| `pet.go` | `Pet` | Escopado por `owner_id` (cliente dono). |
| `consultation.go` | `Consultation` | Anamnese, dieta, dados fisiológicos. |
| `vaccine.go` | `Vaccine` | Lote e próxima dose. |
| `prescription.go` | `Prescription` | Receituário. |
| `catalog_item.go` | `CatalogItem` | Itens de catálogo do vet. |
| `invoice.go` | `Invoice` | Financeiro, com `status`. |
| `appointment.go` | `Appointment` | Agenda (`start_time`). |
| `attachment.go` | `Attachment` | Anexos por `pet_id`. |
| `notification.go` | `Notification` + `NotificationType` | Lembretes de consulta/vacina. |
| `errors.go` | `ValidationError` / `NewValidationError` | Erro de validação de domínio. |

## Validação

Vários DTOs expõem um método `Validate()` que retorna `ValidationError` (via
`NewValidationError`). Exemplos em `user.go`: `RegisterRequest.Validate()` exige
nome/e-mail/senha, senha com ≥ 6 caracteres e confirmação igual.

## Convenções

- Tags `bson` definem o nome do campo no MongoDB e **devem** bater com os
  índices em [`../db/README.md`](../db/README.md) e com os filtros dos
  [`../repositories/README.md`](../repositories/README.md).
- Tags `json` definem o contrato da API; campos sensíveis usam `json:"-"`.
- IDs são `primitive.ObjectID` com `omitempty`.
