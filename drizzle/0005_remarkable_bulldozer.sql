CREATE TYPE "public"."deployment_status" AS ENUM('starting_new', 'health_checking', 'flipping_proxy', 'stopping_old', 'running', 'failed', 'rolled_back');--> statement-breakpoint
CREATE TABLE "deployment" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"build_id" uuid NOT NULL,
	"host_id" uuid,
	"status" "deployment_status" DEFAULT 'starting_new' NOT NULL,
	"started_at" timestamp DEFAULT now() NOT NULL,
	"finished_at" timestamp,
	"triggered_by" text
);
--> statement-breakpoint
CREATE TABLE "env_var" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"key" text NOT NULL,
	"value_ciphertext" text NOT NULL,
	"is_secret" boolean DEFAULT false NOT NULL,
	"updated_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "deployment" ADD CONSTRAINT "deployment_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "deployment" ADD CONSTRAINT "deployment_build_id_build_id_fk" FOREIGN KEY ("build_id") REFERENCES "public"."build"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "deployment" ADD CONSTRAINT "deployment_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE set null ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "env_var" ADD CONSTRAINT "env_var_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "deployment_appId_idx" ON "deployment" USING btree ("app_id");--> statement-breakpoint
CREATE INDEX "deployment_buildId_idx" ON "deployment" USING btree ("build_id");--> statement-breakpoint
CREATE INDEX "deployment_hostId_idx" ON "deployment" USING btree ("host_id");--> statement-breakpoint
CREATE INDEX "envVar_appId_idx" ON "env_var" USING btree ("app_id");