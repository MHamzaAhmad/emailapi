-- Create "users" table
CREATE TABLE "public"."users" (
  "id" text NOT NULL,
  "email" text NOT NULL,
  "name" text NOT NULL,
  "role" text NOT NULL DEFAULT 'member',
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "users_email_key" UNIQUE ("email")
);
-- Create index "idx_users_email" to table: "users"
CREATE INDEX "idx_users_email" ON "public"."users" ("email");
-- Create "api_keys" table
CREATE TABLE "public"."api_keys" (
  "id" text NOT NULL,
  "user_id" text NOT NULL,
  "name" text NOT NULL,
  "key_hash" text NOT NULL,
  "key_prefix" text NOT NULL,
  "scopes" text[] NULL,
  "environment" text NOT NULL DEFAULT 'live',
  "is_active" boolean NOT NULL DEFAULT true,
  "last_used_at" timestamptz NULL,
  "expires_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "api_keys_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_api_keys_is_active" to table: "api_keys"
CREATE INDEX "idx_api_keys_is_active" ON "public"."api_keys" ("is_active") WHERE (is_active = true);
-- Create index "idx_api_keys_key_hash" to table: "api_keys"
CREATE INDEX "idx_api_keys_key_hash" ON "public"."api_keys" ("key_hash");
-- Create index "idx_api_keys_key_prefix" to table: "api_keys"
CREATE INDEX "idx_api_keys_key_prefix" ON "public"."api_keys" ("key_prefix");
-- Create index "idx_api_keys_user_id" to table: "api_keys"
CREATE INDEX "idx_api_keys_user_id" ON "public"."api_keys" ("user_id");
-- Create index "idx_api_keys_environment" to table: "api_keys"
CREATE INDEX "idx_api_keys_environment" ON "public"."api_keys" ("environment");
-- Create "domains" table
CREATE TABLE "public"."domains" (
  "id" text NOT NULL,
  "user_id" text NOT NULL,
  "domain_name" text NOT NULL,
  "status" text NOT NULL DEFAULT 'pending',
  "verified_for_sending" boolean NOT NULL DEFAULT false,
  "dkim_tokens" text[] NULL,
  "dkim_status" text NOT NULL DEFAULT 'pending',
  "mail_from_domain" text NULL,
  "mail_from_status" text NULL,
  "region" text NOT NULL DEFAULT 'us-east-1',
  "last_verified_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "domains_user_id_domain_name_key" UNIQUE ("user_id", "domain_name"),
  CONSTRAINT "domains_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "public"."users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_domains_domain_name" to table: "domains"
CREATE INDEX "idx_domains_domain_name" ON "public"."domains" ("domain_name");
-- Create index "idx_domains_last_verified_at" to table: "domains"
CREATE INDEX "idx_domains_last_verified_at" ON "public"."domains" ("last_verified_at");
-- Create index "idx_domains_status" to table: "domains"
CREATE INDEX "idx_domains_status" ON "public"."domains" ("status");
-- Create index "idx_domains_user_id" to table: "domains"
CREATE INDEX "idx_domains_user_id" ON "public"."domains" ("user_id");
