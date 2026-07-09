CREATE TABLE "agent_release" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"version" text NOT NULL,
	"platform" text NOT NULL,
	"download_url" text NOT NULL,
	"sha256" text NOT NULL,
	"released_at" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE INDEX "agentRelease_platform_releasedAt_idx" ON "agent_release" USING btree ("platform","released_at");