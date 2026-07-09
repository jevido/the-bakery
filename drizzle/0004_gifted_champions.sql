CREATE TYPE "public"."build_status" AS ENUM('queued', 'building', 'succeeded', 'failed');--> statement-breakpoint
CREATE TYPE "public"."source_provider" AS ENUM('github_app');--> statement-breakpoint
CREATE TABLE "app" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" text NOT NULL,
	"name" text NOT NULL,
	"repo_id" uuid,
	"build_context" text DEFAULT '.' NOT NULL,
	"dockerfile_path" text,
	"host_id" uuid,
	"created_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "build" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"repo_id" uuid NOT NULL,
	"commit_sha" text NOT NULL,
	"branch" text NOT NULL,
	"triggered_by" text,
	"status" "build_status" DEFAULT 'queued' NOT NULL,
	"image_ref" text,
	"started_at" timestamp,
	"finished_at" timestamp
);
--> statement-breakpoint
CREATE TABLE "build_log_line" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"build_id" uuid NOT NULL,
	"ts" timestamp DEFAULT now() NOT NULL,
	"line" text NOT NULL
);
--> statement-breakpoint
CREATE TABLE "repo" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"source_id" uuid NOT NULL,
	"full_name" text NOT NULL,
	"default_branch" text NOT NULL
);
--> statement-breakpoint
CREATE TABLE "source" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"organization_id" text NOT NULL,
	"provider" "source_provider" DEFAULT 'github_app' NOT NULL,
	"github_installation_id" text,
	"connected_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "app" ADD CONSTRAINT "app_organization_id_organization_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organization"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "app" ADD CONSTRAINT "app_repo_id_repo_id_fk" FOREIGN KEY ("repo_id") REFERENCES "public"."repo"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "app" ADD CONSTRAINT "app_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "build" ADD CONSTRAINT "build_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "build" ADD CONSTRAINT "build_repo_id_repo_id_fk" FOREIGN KEY ("repo_id") REFERENCES "public"."repo"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "build_log_line" ADD CONSTRAINT "build_log_line_build_id_build_id_fk" FOREIGN KEY ("build_id") REFERENCES "public"."build"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "repo" ADD CONSTRAINT "repo_source_id_source_id_fk" FOREIGN KEY ("source_id") REFERENCES "public"."source"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "source" ADD CONSTRAINT "source_organization_id_organization_id_fk" FOREIGN KEY ("organization_id") REFERENCES "public"."organization"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "app_organizationId_idx" ON "app" USING btree ("organization_id");--> statement-breakpoint
CREATE INDEX "build_appId_idx" ON "build" USING btree ("app_id");--> statement-breakpoint
CREATE INDEX "build_repoId_idx" ON "build" USING btree ("repo_id");--> statement-breakpoint
CREATE INDEX "buildLogLine_buildId_ts_idx" ON "build_log_line" USING btree ("build_id","ts");--> statement-breakpoint
CREATE INDEX "repo_sourceId_idx" ON "repo" USING btree ("source_id");--> statement-breakpoint
CREATE INDEX "source_organizationId_idx" ON "source" USING btree ("organization_id");