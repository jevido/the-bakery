CREATE TABLE "volume" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"host_id" uuid NOT NULL,
	"name" text NOT NULL,
	"mount_path" text NOT NULL,
	"size_bytes" bigint DEFAULT 0 NOT NULL,
	"created_at" timestamp DEFAULT now() NOT NULL,
	"last_reported_at" timestamp,
	CONSTRAINT "volume_appId_name_unique" UNIQUE("app_id","name")
);
--> statement-breakpoint
ALTER TABLE "volume" ADD CONSTRAINT "volume_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "volume" ADD CONSTRAINT "volume_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "volume_appId_idx" ON "volume" USING btree ("app_id");--> statement-breakpoint
CREATE INDEX "volume_hostId_idx" ON "volume" USING btree ("host_id");