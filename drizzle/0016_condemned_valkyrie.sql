ALTER TABLE "host_metric_sample" ADD COLUMN "swap_pct" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "load_avg_1" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "disk_read_bytes_per_sec" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "disk_write_bytes_per_sec" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "net_rx_bytes_per_sec" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "net_tx_bytes_per_sec" real;--> statement-breakpoint
ALTER TABLE "host_metric_sample" ADD COLUMN "uptime_seconds" real;