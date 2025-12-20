env "local" {
  src = "../db/migrations"
  url = "postgres://emailapi:emailapi@localhost:5432/emailapi?sslmode=disable"
  dev = "docker://postgres/16/dev"
}

env "prod" {
  src = "../db/migrations"
  url = env("DATABASE_URL")
}
