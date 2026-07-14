CREATE TABLE "domain" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"hostname" text NOT NULL,
	"is_default_subdomain" boolean DEFAULT false NOT NULL,
	"verified_at" timestamp,
	"created_at" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "domain_hostname_unique" UNIQUE("hostname")
);
--> statement-breakpoint
ALTER TABLE "domain" ADD CONSTRAINT "domain_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "domain_appId_idx" ON "domain" USING btree ("app_id");