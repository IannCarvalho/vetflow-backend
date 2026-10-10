# `services/` — Regras de negócio e tarefas de fundo

[⬆️ Voltar para o README do backend](../README.md)

Camada de lógica de negócio que orquestra múltiplos repositórios. Hoje
concentra o domínio de **notificações**: geração de lembretes (consultas e
vacinas próximas) e o scheduler que dispara e-mails diários.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `notification_service.go` | `NotificationService` — gera e gerencia notificações. |
| `notification_scheduler.go` | `NotificationScheduler` — loop periódico de envio de e-mail. |
| `notification_service_test.go` | Testes do serviço de notificações. |

## `NotificationService`

Interface com:

- `GenerateNotifications(ctx, vetID)` — varre consultas/vacinas próximas e cria
  as notificações devidas. Evita N+1 coletando os `petID` dos lembretes
  realmente devidos antes de buscar os pets.
- `GetNotifications(ctx, vetID)` — lista as notificações do vet.
- `MarkAsRead` / `Dismiss` — transições de estado por `notificationID`.

Depende dos repositórios de `notification`, `consultation`, `vaccine` e `pet`
(ver [`../repositories/README.md`](../repositories/README.md)), injetados via
`NewNotificationService(...)`.

## `NotificationScheduler`

Iniciado por `cmd/main.go` **apenas** quando
`EMAIL_NOTIFICATIONS_ENABLED=true`. Roda em intervalo fixo (24h), gera as
notificações pendentes e envia os e-mails usando o `Mailer` de
[`../helpers/README.md`](../helpers/README.md). É não bloqueante e respeita o
contexto de shutdown do processo.

## Quem consome

`NotificationService` é injetado no `NotificationController`
([`../controllers/README.md`](../controllers/README.md)); o scheduler é iniciado
no bootstrap em [`../cmd/README.md`](../cmd/README.md).
