CREATE TYPE "public"."host_status" AS ENUM('pending', 'online', 'offline');--> statement-breakpoint
CREATE TABLE "host" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" text NOT NULL,
	"name" text NOT NULL,
	"location" text,
	"spec" text,
	"token_hash" text NOT NULL,
	"token_last_four" text,
	"status" "host_status" DEFAULT 'pending' NOT NULL,
	"agent_version" text,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"last_seen_at" timestamp,
	"revoked_at" timestamp
);
--> statement-breakpoint
CREATE TABLE "host_metric_sample" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"host_id" uuid NOT NULL,
	"ts" timestamp DEFAULT now() NOT NULL,
	"cpu_pct" real,
	"mem_pct" real,
	"disk_pct" real,
	"podman_version" text,
	"container_count" integer
);
--> statement-breakpoint
ALTER TABLE "host" ADD CONSTRAINT "host_organization_id_organization_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organization"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD CONSTRAINT "host_metric_sample_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "host_organizationId_idx" ON "host" USING btree ("organization_id");--> statement-breakpoint
CREATE INDEX "hostMetricSample_hostId_ts_idx" ON "host_metric_sample" USING btree ("host_id","ts");