// Atlas configuration for declarative schema management
// Uses schema files as source of truth and generates migrations via diff

env "local" {
  // Source of truth: declarative schema files
  src = "file://db/postgres/schema"
  
  // Target database URL
  url = "postgres://emailapi:emailapi@localhost:5432/emailapi?sslmode=disable"
  
  // Dev database for diffing (spins up ephemeral container)
  dev = "docker://postgres/18/dev"
  
  // Migration directory
  migration {
    dir = "file://db/postgres/migrations"
  }
}

env "prod" {
  // Source of truth: declarative schema files
  src = "file://db/postgres/schema"
  
  // Target database URL from environment
  url = env("DATABASE_URL")
  
  // Migration directory
  migration {
    dir = "file://db/postgres/migrations"
  }
}
