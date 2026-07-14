CREATE TABLE "app_log_line" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"host_id" uuid NOT NULL,
	"ts" timestamp DEFAULT now() NOT NULL,
	"level" text,
	"message" text NOT NULL
);
--> statement-breakpoint
ALTER TABLE "app_log_line" ADD CONSTRAINT "app_log_line_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "app_log_line" ADD CONSTRAINT "app_log_line_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "appLogLine_appId_ts_idx" ON "app_log_line" USING btree ("app_id","ts");