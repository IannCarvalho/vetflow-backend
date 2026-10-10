# `repositories/` — Camada de acesso a dados

[⬆️ Voltar para o README do backend](../README.md)

Abstrai todo o acesso ao MongoDB atrás de **interfaces** (repository pattern).
Os controllers dependem das interfaces, não das implementações concretas — o
que mantém o código testável (fakes em `controllers/fakes_test.go`) e desacopla
o domínio do driver do banco.

## Estrutura

| Arquivo | Conteúdo |
| --- | --- |
| `interfaces.go` | Todas as interfaces de repositório (`ClientRepository`, `PetRepository`, ...). |
| `<entidade>_repository.go` | Implementação MongoDB de cada interface. |

Repositórios: `client`, `pet`, `consultation`, `vaccine`, `user`,
`prescription`, `catalog_item`, `invoice`, `appointment`, `attachment`,
`notification`.

## Padrões das interfaces

Cada repositório expõe um CRUD assíncrono recebendo `context.Context`:

```go
Create(ctx, *models.X) error
FindByID(ctx, id primitive.ObjectID) (*models.X, error)
FindAll(ctx) ([]*models.X, error)
Update(ctx, id, *models.X) error
Delete(ctx, id) error
```

E métodos escopados por veterinário, essenciais para o isolamento multi-tenant:

- `FindByVetID(ctx, vetID)` — lista apenas os registros do vet autenticado.
- `CountByVetID` / `CountByVetIDInRange` / `CountByVetIDAndStatus` /
  `CountByVetIDNextDoseInRange` — contagens do dashboard, cobertas pelos índices
  compostos declarados em [`../db/README.md`](../db/README.md).
- Métodos de cascata como `DeleteByPetID` / `DeleteByClientID`.

> `(nil, nil)` em `FindByID` significa "não encontrado" (não é erro); os
> controllers tratam esse caso explicitamente.

## Quem consome

Instanciados em `routes.NewRouter` e injetados nos
[`../controllers/README.md`](../controllers/README.md) e nos
[`../services/README.md`](../services/README.md).
