# HEALTH Backend

A backend service built with Go + Gin that acts as a middleware between hospital staff and patient data. Staff can only search patients from their own hospital, the system enforces this via JWT claims tied to the hospital at login.

When a search includes a `national_id` or `passport_id`, the service calls the external HIS API first and upserts the result into the local database. The local database is always queried last and is the source of the response. If no ID is provided, the HIS API is skipped and only the local database is searched.


## Stack

- Go 1.26 + Gin
- PostgreSQL (via GORM)
- Docker + Nginx
- JWT auth (24h expiry)
- SQLite in-memory (tests only)


## Environment Variables

This project uses environment variables for configuration.

A sample file .env.example is provided.

### Setup

Copy the example file:

```bash
cp .env.example .env
```

## Project Structure

```
health-backend/
├── agnoshealth/
│   ├── config/
│   │   ├── config.go       # env-based config
│   │   └── database.go     # postgres connection + auto-migrate
│   ├── handler/
│   │   ├── patient.go      # GET+POST /patient/search
│   │   ├── patient_test.go
│   │   ├── staff.go        # POST /staff/create, /staff/login
│   │   └── staff_test.go
│   ├── middleware/
│   │   └── auth.go         # JWT middleware
│   └── model/
│       ├── hospital.go
│       ├── patient.go
│       └── staff.go
├── external/
│   └── hospital.go         # HIS API client
├── server/
│   └── main.go
├── docker-compose.yml
├── Dockerfile
├── nginx.conf
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Running

```bash
docker compose up --build
```

API available at `http://localhost` (Nginx proxies to port 80).

## Running Tests

Tests use an in-memory SQLite database so no external services needed.

```bash
CGO_ENABLED=1 go test ./agnoshealth/handler/... -v
```

<details>
<summary>Test output</summary>

```
=== RUN   TestSearchPatient_NoAuth
--- PASS: TestSearchPatient_NoAuth (0.00s)
=== RUN   TestSearchPatient_Success
--- PASS: TestSearchPatient_Success (0.00s)
=== RUN   TestSearchPatient_HospitalIsolation
--- PASS: TestSearchPatient_HospitalIsolation (0.00s)
=== RUN   TestSearchPatient_WithNationalIDCallsHIS
--- PASS: TestSearchPatient_WithNationalIDCallsHIS (0.00s)
=== RUN   TestCreateStaff_Success
--- PASS: TestCreateStaff_Success (0.08s)
=== RUN   TestCreateStaff_Duplicate
--- PASS: TestCreateStaff_Duplicate (0.13s)
=== RUN   TestCreateStaff_MissingFields
--- PASS: TestCreateStaff_MissingFields (0.00s)
=== RUN   TestLogin_Success
--- PASS: TestLogin_Success (0.13s)
=== RUN   TestLogin_WrongPassword
--- PASS: TestLogin_WrongPassword (0.13s)
=== RUN   TestLogin_WrongHospital
--- PASS: TestLogin_WrongHospital (0.07s)
PASS
ok  	health-backend/agnoshealth/handler	1.884s
```
</details>

## Seeding Data

The external HIS API (`hospital-a.api.co.th`) isn't reachable, so we insert data directly for manual testing.

**Hospitals must be seeded first** — `/staff/create` looks up the hospital by name and returns 400 if it doesn't exist.

```bash
docker compose exec db psql -U postgres -d agnos_health_db
```

```sql
-- seed hospitals first
INSERT INTO hospitals (name, created_at, updated_at)
VALUES
  ('Mission Hospital', NOW(), NOW()),
  ('Bangkok Hospital', NOW(), NOW());

-- seed patients (first two belong to Mission Hospital, last one to Bangkok Hospital)
INSERT INTO patients (
  created_at, updated_at,
  first_name_th, middle_name_th, last_name_th,
  first_name_en, middle_name_en, last_name_en,
  date_of_birth, patient_hn, national_id, passport_id,
  phone_number, email, gender, hospital_id
)
VALUES
  (NOW(), NOW(), 'อนันต์', 'ชัยวัฒน์', 'เกียรติรุ่งเรือง',
   'Anan', 'Chaiwat', 'Kiatrungrueang',
   '1990-05-15', 'HN-00123', '1234567890123', NULL,
   '0812345678', 'anan@example.com', 'M', 1),

  (NOW(), NOW(), 'นรินทร์', 'พงษ์ศักดิ์', 'สุวรรณรักษ์',
   'Narin', 'Pongsak', 'Suwanrak',
   '1995-08-20', 'HN-00124', '9876543210987', 'BK3495THAI',
   '0898765432', 'narin@example.com', 'F', 1),

  (NOW(), NOW(), 'ปริญญา', 'วิชัย', 'ธนภูมิ',
   'Parinya', 'Vichai', 'Thanaphum',
   '1985-01-10', 'HN-00200', '1111111111111', 'RM3452THAW',
   '0811111111', 'parinya@example.com', 'M', 2);

\q
```

## API

| Endpoint          | Method   | Auth | Description                          |
|-------------------|----------|------|--------------------------------------|
| `/health`         | GET      | No   | Health check                         |
| `/staff/create`   | POST     | No   | Register a staff account             |
| `/staff/login`    | POST     | No   | Login, returns JWT (24h)             |
| `/patient/search` | GET/POST | Yes  | Search patients (hospital-scoped, max 50 results) |

### Create staff

Hospital must already exist in the database.

```bash
curl -X POST http://localhost/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username": "palatip", "password": "pass@123", "hospital": "Mission Hospital"}'
```

**Created**

```json
{"id": 1, "username": "palatip", "hospital_id": 1, "hospital": "Mission Hospital", "status": "success"}
```

**Conflict** (duplicate username + hospital)

```json
{ "error": "staff username already exists", "status": "failed" }
```

**Bad Request** (missing fields)

```json
{ "errors": {"username": "username is required", "password": "password is required", "hospital": "hospital is required", "status": "failed"}
}
```

### Login

```bash
curl -X POST http://localhost/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username": "palatip", "password": "pass@123", "hospital": "Mission Hospital"}'
```

**Successful**

```json
{"hospital": "Mission Hospital", "token": "<jwt>", "status": "success"}
```

**Unauthorized** (wrong password or wrong hospital)

```json
{ "error": "invalid credentials", "status": "failed" }
```

Copy the token — use it as `Authorization: Bearer <token>` for patient search.

### Search patients

GET with query params:
```bash
curl "http://localhost/patient/search?national_id=1234567890123" \
  -H "Authorization: Bearer <token>"
```

**OK** — patient found
```json
{"count":1, "patients":[{ "id":1, "first_name_th":"อนันต์", "middle_name_th":"ชัยวัฒน์", "last_name_th":"เกียรติรุ่งเรือง", "first_name_en":"Anan", "middle_name_en":"Chaiwat", "last_name_en":"Kiatrungrueang", "date_of_birth":"1990-05-15T00:00:00Z", "patient_hn":"HN-00123", "national_id":"1234567890123", "passport_id":"", "PhoneNumber":"0812345678", "Email":"somchai@example.com", "gender":"M", "HospitalID":1, "hospital":{"id":0,"name":"","created_at":"0001-01-01T00:00:00Z", "updated_at":"0001-01-01T00:00:00Z"}, "created_at":"2026-03-28T14:54:37.101148Z","updated_at":"2026-03-28T14:54:37.101148Z"}],"status":"success"}

```

**OK** — no matching patients (different hospital or no match)
```json
{ "count": 0, "patients": [], "status": "success" }
```

**Unauthorized** (missing or invalid token)
```json
{ "error": ["invalid or expired token", "authorization header required", "invalid authorization header format"],  "status": "failed"}
```

POST with JSON body:
```bash
curl -X POST http://localhost/patient/search \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"first_name": "Anan"}'
```

**OK**
```json
{"count":1,  "patients":[{"id":2, "first_name_th":"นรินทร์", "middle_name_th":"พงษ์ศักดิ์", "last_name_th":"สุวรรณรักษ์", "first_name_en":"Narin", "middle_name_en":"Pongsak","last_name_en":"Suwanrak", "date_of_birth":"1995-08-20T00:00:00Z", "patient_hn":"HN-00124", "national_id":"9876543210987", "passport_id":"BK3495THAI", "PhoneNumber":"0898765432", "Email":"somying@example.com", "gender":"F", "HospitalID":1, "hospital":{"id":0,"name":"","created_at":"0001-01-01T00:00:00Z", "updated_at":"0001-01-01T00:00:00Z"}, "created_at":"2026-03-28T14:54:37.101148Z", "updated_at":"2026-03-28T14:54:37.101148Z"}], "status":"success"}


```

**200 OK** — no matching patients (different hospital or no match)
```json
{"count": 0, "patients": [], "status": "success"}
```

Supported filters (all optional): `national_id`, `passport_id`, `first_name`, `middle_name`, `last_name`, `date_of_birth`, `phone_number`, `email`.

Name filters use partial, case-insensitive matching (ILIKE) and check both TH and EN fields.

### Hospital isolation

Staff only see patients from their own hospital. Logging in as Bangkok Hospital staff and searching by a `national_id` that belongs to a Mission Hospital patient returns empty.

```bash
# Create and login as Bangkok Hospital staff
curl -X POST http://localhost/staff/create \
  -H "Content-Type: application/json" \
  -d '{"username": "somchai", "password": "pass@123", "hospital": "Bangkok Hospital"}'

curl -X POST http://localhost/staff/login \
  -H "Content-Type: application/json" \
  -d '{"username": "somchai", "password": "pass@123", "hospital": "Bangkok Hospital"}'

# This returns empty even though 1234567890123 exists — it belongs to Mission Hospital
curl "http://localhost/patient/search?national_id=1234567890123" \
  -H "Authorization: Bearer <bangkok_token>"
```

```json
{"count": 0, "patients": [], "status": "success"}
```

## ER Diagram
```
+-----------------------+              +-----------------------------------+
|       hospitals       |              |          patients                 |
+-----------------------+              +-----------------------------------+
| id (PK)               |              | id (PK)                           |
| name (UNIQUE)         |              | first_name_th  (string)           |
| created_at (timestamp)|----|         | middle_name_th (string)           |
| updated_at (timestamp)|    |         | last_name_th   (string)           |
| deleted_at (timestamp)|    |         | first_name_en  (string)           |
+-----------------------+    |         | middle_name_en (string)           |
            |                |         | last_name_en   (string)           |
            |                |         | date_of_birth  (date)             |
            |                |         | patient_hn     (string)           |
            |                |         | national_id    (string) (UNIQUE*) |
            |                |         | passport_id    (string) (UNIQUE*) |
            |                |         | phone_number   (string)           |
            |                |         | email          (string)           |
            |                |         | gender         (CHAR 1)           |
            |                |--[1:N]--| hospital_id    (FK, INDEX)        |
            |                          | created_at     (timestamp)        |
            |                          | updated_at     (timestamp)        |
            |                          | deleted_at     (timestamp)        |
            |                          +-----------------------------------+
            |
            |          +----------------------------+
            |          |       staffs               |
            |          +----------------------------+
            |          | id (PK)                    |
            |          | username (string) (UNIQUE*)|
            |          | password  (string)         |
            | --[1:N]--| hospital_id (FK)           |
                       | created_at (timestamp)     |
                       | updated_at (timestamp)     |
                       | deleted_at (timestamp)     |
                       +----------------------------+
```

