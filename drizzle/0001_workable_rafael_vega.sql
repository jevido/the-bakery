ALTER TABLE "organization" ADD COLUMN "invite_code" text NOT NULL;--> statement-breakpoint
ALTER TABLE "organization" ADD COLUMN "color" text NOT NULL;--> statement-breakpoint
ALTER TABLE "organization" ADD CONSTRAINT "organization_invite_code_unique" UNIQUE("invite_code");