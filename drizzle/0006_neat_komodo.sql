CREATE TYPE "public"."host_command_status" AS ENUM('pending', 'delivered', 'succeeded', 'failed');--> statement-breakpoint
CREATE TYPE "public"."host_command_type" AS ENUM('deploy', 'stop', 'restart');--> statement-breakpoint
CREATE TABLE "host_command" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"host_id" uuid NOT NULL,
	"deployment_id" uuid NOT NULL,
	"type" "host_command_type" NOT NULL,
	"payload" jsonb NOT NULL,
	"status" "host_command_status" DEFAULT 'pending' NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"delivered_at" timestamp,
	"completed_at" timestamp,
	"error_message" text
);
--> statement-breakpoint
ALTER TABLE "host_command" ADD CONSTRAINT "host_command_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "host_command" ADD CONSTRAINT "host_command_deployment_id_deployment_id_fk" FOREIGN KEY ("deployment_id") REFERENCES "public"."deployment"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "hostCommand_hostId_status_idx" ON "host_command" USING btree ("host_id","status");--> statement-breakpoint
CREATE INDEX "hostCommand_deploymentId_idx" ON "host_command" USING btree ("deployment_id");