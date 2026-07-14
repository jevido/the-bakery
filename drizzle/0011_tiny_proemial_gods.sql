CREATE TABLE "app_metric_sample" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"app_id" uuid NOT NULL,
	"host_id" uuid NOT NULL,
	"ts" timestamp DEFAULT now() NOT NULL,
	"cpu_pct" real,
	"mem_bytes" bigint
);
--> statement-breakpoint
ALTER TABLE "app_metric_sample" ADD CONSTRAINT "app_metric_sample_app_id_app_id_fk" FOREIGN KEY ("app_id") REFERENCES "public"."app"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "app_metric_sample" ADD CONSTRAINT "app_metric_sample_host_id_host_id_fk" FOREIGN KEY ("host_id") REFERENCES "public"."host"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
CREATE INDEX "appMetricSample_appId_ts_idx" ON "app_metric_sample" USING btree ("app_id","ts");