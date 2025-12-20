-- Modify "api_keys" table
ALTER TABLE "public"."api_keys" ADD COLUMN "environment" text NOT NULL DEFAULT 'live';
-- Create index "idx_api_keys_environment" to table: "api_keys"
CREATE INDEX "idx_api_keys_environment" ON "public"."api_keys" ("environment");
