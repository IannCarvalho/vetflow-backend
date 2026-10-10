# `helpers/` — Utilitários transversais

[⬆️ Voltar para o README do backend](../README.md)

Funções puras e sem estado reutilizadas por controllers e services:
autenticação JWT, hashing de senha, validações brasileiras, e-mail e imagens.

## Arquivos

| Arquivo | Responsabilidade |
| --- | --- |
| `jwt.go` | Geração/validação de JWT (`GenerateToken`, `ParseToken`, `Claims`). |
| `password.go` | Hash e verificação de senha. |
| `validation.go` | Validadores BR: CPF, e-mail, telefone, CEP. |
| `email.go` | `Mailer` SMTP para e-mails de notificação. |
| `image.go` | Tratamento/validação de imagens (ex.: foto de perfil base64). |
| `*_test.go` | Testes unitários de cada utilitário. |

## JWT (`jwt.go`)

- `GenerateToken(userID)` emite um HS256 com TTL de 24h.
- `ParseToken(tokenString)` valida assinatura (exige HMAC), expiração e claims.
- **Fail-closed**: `JWT_SECRET` ausente faz assinar/verificar retornar erro —
  nunca há segredo padrão. Alinhado com `config.Validate()`.
- `Claims` embute `UserID`; a `AuthMiddleware` em
  [`../routes`](../routes/README.md) lê esse campo e injeta no contexto.

## Validações (`validation.go`)

Validadores de documentos/campos brasileiros, aceitando valores com ou sem
máscara:

- `IsValidCPF` — 11 dígitos + dígitos verificadores oficiais.
- `IsValidEmail` — `net/mail`, rejeita formatos com display name.
- `IsValidBRPhone` — 10 (fixo) ou 11 (celular) dígitos.
- `IsValidCEP` — exatamente 8 dígitos.

## E-mail (`email.go`)

`NewMailer(EmailConfig{...})` cria o `Mailer` SMTP. Quando `SMTPHost` está
vazio, roda em modo "log only" (nenhum e-mail real é enviado). Consumido pelo
`NotificationScheduler` em [`../services`](../services/README.md).
