# VetFlow - Veterinary Management System API

A production-ready REST API for managing veterinary clinics, built with Go, MongoDB, and following SOLID principles and MVC architecture.

## Features

- **Client Management**: Register and manage veterinary clinic clients
- **Pet Management**: Track pets with detailed information (species, breed, age, weight)
- **Consultation Records**: Store comprehensive consultation notes with anamnesis, diet, and physiological data
- **Vaccine Tracking**: Maintain vaccine records with batch information and next dose dates
- **RESTful API**: Clean, intuitive endpoints following REST conventions
- **MongoDB Integration**: Scalable NoSQL database for flexible data storage
- **Docker Support**: Easy deployment with Docker and Docker Compose
- **SOLID Principles**: Well-structured, maintainable, and extensible codebase

## Architecture

### Design Pattern
- **MVC (Model-View-Controller)** adapted for REST APIs
- Dependency Injection for loose coupling
- Repository pattern for data access abstraction
- Interface-based design for testability

### Project Structure

```
veterinary-api/
├── cmd/
│   └── main.go                      # Entry point
├── config/
│   └── config.go                    # Configuration management
├── db/
│   └── mongo.go                     # MongoDB connection
├── models/
│   ├── client.go
│   ├── pet.go
│   ├── consultation.go
│   └── vaccine.go
├── repositories/
│   ├── interfaces.go                # Repository interfaces
│   ├── client_repository.go
│   ├── pet_repository.go
│   ├── consultation_repository.go
│   └── vaccine_repository.go
├── controllers/
│   ├── client_controller.go
│   ├── pet_controller.go
│   ├── consultation_controller.go
│   ├── vaccine_controller.go
│   └── helpers.go
├── routes/
│   └── router.go                    # Route definitions
├── docker-compose.yml
├── Dockerfile
├── .env.example
├── go.mod
└── README.md
```

## Prerequisites

- Go 1.26 or higher
- Docker and Docker Compose (optional, for containerized deployment)
- MongoDB 7.0+ (if running locally)

## Setup

### 1. Navigate to Backend Directory

**Windows:**
```powershell
cd backend
```

**Mac/Linux:**
```bash
cd backend
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Configure Environment

**Windows (PowerShell):**
```powershell
copy .env.example .env
# Edit .env with your settings
```

**Mac/Linux:**
```bash
cp .env.example .env
nano .env  # or your preferred editor
```

### 4. Run with Docker Compose (Recommended - All Platforms)

From the root directory:

```powershell
# Windows / Mac / Linux
docker-compose up -d

# The API will be available at http://localhost:8080
```

### 5. Run Locally

**Start MongoDB first:**

```powershell
docker run -d -p 27017:27017 `
  -e MONGO_INITDB_ROOT_USERNAME=admin `
  -e MONGO_INITDB_ROOT_PASSWORD=admin `
  mongo:7.0
```

**Update `.env` file:**
```env
MONGO_URI=mongodb://admin:admin@localhost:27017
MONGO_DB=vetflow
PORT=8080
```

**Run the application:**

```powershell
# Windows
go run ./cmd/main.go

# Mac/Linux
go run ./cmd/main.go
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Health Check
- `GET /health` - Check API status

### Clients
- `POST /clients` - Create a new client
- `GET /clients` - List all clients
- `GET /clients/{id}` - Get client by ID
- `PUT /clients/{id}` - Update client
- `DELETE /clients/{id}` - Delete client (cascades to pets, consultations, vaccines)

### Pets
- `POST /pets` - Create a new pet
- `GET /pets` - List all pets
- `GET /pets?owner_id={clientId}` - List pets by owner
- `GET /pets/{id}` - Get pet by ID
- `PUT /pets/{id}` - Update pet
- `DELETE /pets/{id}` - Delete pet (cascades to consultations, vaccines)

### Consultations
- `POST /consultations` - Create a new consultation
- `GET /consultations/{id}` - Get consultation by ID
- `GET /consultations?pet_id={petId}` - List consultations by pet
- `PUT /consultations/{id}` - Update consultation
- `DELETE /consultations/{id}` - Delete consultation

### Vaccines
- `POST /vaccines` - Create a new vaccine record
- `GET /vaccines/{id}` - Get vaccine by ID
- `GET /vaccines?pet_id={petId}` - List vaccines by pet
- `PUT /vaccines/{id}` - Update vaccine record
- `DELETE /vaccines/{id}` - Delete vaccine record

## API Examples

### Create a Client

**Mac/Linux (curl):**
```bash
curl -X POST http://localhost:8080/clients \
  -H "Content-Type: application/json" \
  -d '{
    "full_name": "João Silva",
    "cpf": "123.456.789-00",
    "phone": "(11) 98765-4321",
    "address": {
      "street": "Rua das Flores",
      "number": "123",
      "complement": "Apt 45",
      "zip_code": "01234-567"
    }
  }'
```

**Windows (PowerShell):**
```powershell
$body = @{
    full_name = "João Silva"
    cpf = "123.456.789-00"
    phone = "(11) 98765-4321"
    address = @{
        street = "Rua das Flores"
        number = "123"
        complement = "Apt 45"
        zip_code = "01234-567"
    }
} | ConvertTo-Json

Invoke-WebRequest -Uri http://localhost:8080/clients `
  -Method POST `
  -Headers @{"Content-Type"="application/json"} `
  -Body $body
```

### Create a Pet

**Mac/Linux (curl):**
```bash
curl -X POST http://localhost:8080/pets \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Rex",
    "species": "dog",
    "breed": "Labrador",
    "age": 3,
    "age_unit": "years",
    "sex": "male",
    "neutered": true,
    "weight": 32.5,
    "owner_id": "your_client_id_here"
  }'
```

**Windows (PowerShell):**
```powershell
$body = @{
    name = "Rex"
    species = "dog"
    breed = "Labrador"
    age = 3
    age_unit = "years"
    sex = "male"
    neutered = $true
    weight = 32.5
    owner_id = "your_client_id_here"
} | ConvertTo-Json

Invoke-WebRequest -Uri http://localhost:8080/pets `
  -Method POST `
  -Headers @{"Content-Type"="application/json"} `
  -Body $body
```

### Create a Consultation

```bash
curl -X POST http://localhost:8080/consultations \
  -H "Content-Type: application/json" \
  -d '{
    "pet_id": "your_pet_id_here",
    "vet_name": "Dr. Maria Santos",
    "date": "2024-05-04T14:30:00Z",
    "anamnesis": {
      "main_complaint": "Cough and sneezing",
      "evolution_time": "2 weeks",
      "symptoms_worsened": false,
      "symptom_frequency": "occasional",
      "behavior_change": false,
      "pain": false,
      "happened_before": false,
      "prior_vet_visit": true
    },
    "diet": {
      "food_type": "dry kibble",
      "amount_and_frequency": "2 cups twice daily",
      "diet_changed_recently": false,
      "normal_water_intake": true
    },
    "physiological": {
      "normal_urine_feces": true,
      "vomiting": false,
      "respiratory_issues": true
    },
    "summary": {
      "main_complaint": "Upper respiratory infection",
      "evolution_time": "2 weeks",
      "feeding": "Normal appetite",
      "feces_urine": "Normal",
      "vaccines_and_deworming": "Up to date",
      "medications": "None current",
      "environment_and_contacts": "Multiple pets in household"
    }
  }'
```

### Record a Vaccine

```bash
curl -X POST http://localhost:8080/vaccines \
  -H "Content-Type: application/json" \
  -d '{
    "pet_id": "your_pet_id_here",
    "vaccine_name": "Rabies",
    "batch": "BATCH2024001",
    "manufacturer": "MSD",
    "date_applied": "2024-05-04T10:00:00Z",
    "next_dose_date": "2025-05-04T10:00:00Z",
    "vet_name": "Dr. Maria Santos",
    "notes": "No adverse reactions"
  }'
```

## Error Handling

All errors follow a consistent JSON format:

```json
{
  "error": "error message description"
}
```

### HTTP Status Codes
- `200 OK` - Successful request
- `201 Created` - Resource successfully created
- `204 No Content` - Successful deletion
- `400 Bad Request` - Invalid input or malformed request
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server-side error

## Data Models

### Client
- `id`: MongoDB ObjectID (auto-generated)
- `full_name`: string (required)
- `cpf`: string (required, unique)
- `phone`: string (required)
- `address`: object (street, number, complement, zip_code)
- `created_at`: timestamp
- `updated_at`: timestamp

### Pet
- `id`: MongoDB ObjectID (auto-generated)
- `name`: string (required)
- `species`: string (required) - e.g., "dog", "cat"
- `breed`: string
- `age`: integer
- `age_unit`: string - "months" or "years"
- `sex`: string - "male" or "female"
- `neutered`: boolean
- `weight`: float64 (kg)
- `owner_id`: MongoDB ObjectID (required) - references Client
- `created_at`: timestamp
- `updated_at`: timestamp

### Consultation
- `id`: MongoDB ObjectID (auto-generated)
- `pet_id`: MongoDB ObjectID (required) - references Pet
- `vet_name`: string
- `date`: timestamp
- `anamnesis`: object - detailed medical history
- `diet`: object - dietary information
- `physiological`: object - physiological observations
- `summary`: object - consultation summary
- `created_at`: timestamp

### Vaccine
- `id`: MongoDB ObjectID (auto-generated)
- `pet_id`: MongoDB ObjectID (required) - references Pet
- `vaccine_name`: string (required)
- `batch`: string
- `manufacturer`: string
- `date_applied`: timestamp
- `next_dose_date`: timestamp
- `vet_name`: string
- `notes`: string
- `created_at`: timestamp

## SOLID Principles Applied

### Single Responsibility (S)
Each file/struct handles one concern:
- Models: Data structure and validation
- Repositories: Data access
- Controllers: HTTP request handling
- Routes: Endpoint registration

### Open/Closed (O)
- Interfaces define contracts
- New repositories can be added without modifying existing code
- Controllers depend on interfaces, not implementations

### Liskov Substitution (L)
- All repository implementations are interchangeable
- Concrete repositories properly implement their interfaces

### Interface Segregation (I)
- Repository interfaces are focused and specific
- ClientRepository doesn't include pet methods
- Controllers only depend on needed functionality

### Dependency Inversion (D)
- Controllers depend on repository interfaces, not concrete types
- Dependency injection via constructor
- Easy to swap implementations for testing

## Development

### Running Tests

```bash
go test ./...
```

### Build Locally

```bash
# Build binary
go build -o vetflow ./cmd/main.go
```

### Windows Development Setup

**Install Go (Windows):**
1. Download from https://golang.org/dl/
2. Run the installer
3. Restart your terminal
4. Verify: `go version`

**Install MongoDB locally (Windows):**
```powershell
# Using Docker (recommended)
docker run -d -p 27017:27017 `
  -e MONGO_INITDB_ROOT_USERNAME=admin `
  -e MONGO_INITDB_ROOT_PASSWORD=admin `
  --name vetflow-mongo `
  mongo:7.0

# Or download from https://www.mongodb.com/try/download/community
```

**Code Formatting and Linting:**
```powershell
# Format code
go fmt ./...

# Lint with golangci-lint (requires installation)
golangci-lint run ./...

# Or simpler linting
go vet ./...
```

## Building

**Build binary:**

```powershell
# Windows
go build -o vetflow.exe ./cmd/main.go

# Mac/Linux
go build -o vetflow ./cmd/main.go
```

**Build Docker image (all platforms):**

From the root directory:
```powershell
docker build -t vetflow:latest -f backend/Dockerfile .
```

## Database Indexes

For optimal performance, create the following indexes:

```javascript
// Clients collection
db.clients.createIndex({ cpf: 1 }, { unique: true })

// Pets collection
db.pets.createIndex({ owner_id: 1 })

// Consultations collection
db.consultations.createIndex({ pet_id: 1 })

// Vaccines collection
db.vaccines.createIndex({ pet_id: 1 })
```

## License

MIT License

## Support

For issues or questions, please open an issue in the repository.

---

**Built with Go, MongoDB, and ❤️ for veterinary professionals.**
