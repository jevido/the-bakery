package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	loopbackContainerName = "bakery-bootstrap"
	loopbackPort          = 3000

	defaultSourceRepoURL = "https://github.com/jevido/the-bakery.git"
	sourceRepoRef        = "main"

	// selfImageBuildOrg/App name the one image-pointing build/app row
	// bootstrap-api's +server.ts creates directly (no repo/source) — see
	// task 12 (Phase 08)'s registry.go loopback-registry reasoning for why
	// this specific image lives at 127.0.0.1, not registry.<domain>.
	selfImageBuildOrg = "bakery"
	selfImageBuildApp = "bakery-control-plane"
)

// cmdBootstrap closes this phase's chicken-and-egg loop: it provisions
// Postgres, runs migrations and starts the control-plane image bound to
// 127.0.0.1 only, drives that instance's own bootstrap API to create the
// first admin/guild/host/app, then hands off into cmdJoin's logic so the
// control plane ends up running as a real Bakery-managed deployment of
// itself instead of the manually-started loopback container.
func cmdBootstrap(args []string) {
	fs := flag.NewFlagSet("bakery bootstrap", flag.ExitOnError)
	domain := fs.String("domain", "", "public domain this instance will be reachable at (required)")
	adminEmail := fs.String("admin-email", "", "admin account email (prompted if omitted)")
	adminPassword := fs.String("admin-password", "", "admin account password (prompted if omitted)")
	guildName := fs.String("guild-name", "Bakery", "name of the first guild created")
	image := fs.String("image", "", "control-plane container image to deploy (built from --source locally if omitted)")
	source := fs.String("source", defaultSourceRepoURL, "git URL to build the control-plane image from, when --image is omitted")
	fs.Parse(args)

	if *domain == "" {
		fmt.Fprintln(os.Stderr, "Usage: bakery bootstrap --domain=<domain> [--admin-email=...] [--admin-password=...] [--image=...] [--source=...]")
		os.Exit(2)
	}

	if os.Geteuid() == 0 {
		reexecAsBakeryUser(os.Args[1:])
		return
	}
	ensureUserSessionEnv()

	if *adminEmail == "" {
		*adminEmail = promptLine("Admin email: ")
	}
	if *adminPassword == "" {
		*adminPassword = promptLine("Admin password (min 8 characters): ")
	}
	if *adminEmail == "" || *adminPassword == "" {
		bootstrapFail("validating input", fmt.Errorf("admin email and password are both required"))
	}

	// Read from this process's own environment (e.g. a .env the operator
	// sourced before running bootstrap) rather than a new flag — these are
	// long, sometimes multi-line values (a PEM private key) that don't fit
	// the CLI-flag/interactive-prompt shape the admin email/password use.
	//
	// Found live: these were originally treated as optional here — self-
	// hosting "still completes, just with Connect GitHub non-functional
	// until set later." That's wrong. SvelteKit's `defineEnvVars`
	// (src/env.ts) declares all six as required for the app to boot *at
	// all* — GITHUB_CLIENT_ID/_SECRET (GitHub OAuth login) were never even
	// read here before this fix, so the real, post-migration control-plane
	// container crash-looped on startup with no visible error beyond a
	// generic 2-minute "connection refused" from the health-wait loop
	// outside it. Failing fast here, before Postgres/registry/the image
	// build even start, turns a silent ~5+ minute dead end into an
	// immediate, actionable error.
	githubClientID := os.Getenv("GITHUB_CLIENT_ID")
	githubClientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	githubAppID := os.Getenv("GITHUB_APP_ID")
	githubAppSlug := os.Getenv("GITHUB_APP_SLUG")
	githubAppPrivateKey := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if missing := missingRequiredGithubEnvVars(map[string]string{
		"GITHUB_CLIENT_ID":       githubClientID,
		"GITHUB_CLIENT_SECRET":   githubClientSecret,
		"GITHUB_APP_ID":          githubAppID,
		"GITHUB_APP_SLUG":        githubAppSlug,
		"GITHUB_APP_PRIVATE_KEY": githubAppPrivateKey,
		"GITHUB_WEBHOOK_SECRET":  githubWebhookSecret,
	}); len(missing) > 0 {
		bootstrapFail("validating input", fmt.Errorf(
			"the control plane can't start without these set in this process's own environment (e.g. `set -a; source .env; set +a` before running bootstrap): %s",
			strings.Join(missing, ", "),
		))
	}

	fmt.Printf("Checking that %s resolves to this box...\n", *domain)
	if err := checkDNSPointsHere(*domain); err != nil {
		bootstrapFail("checking DNS", err)
	}
	fmt.Println("DNS OK.")

	home, err := os.UserHomeDir()
	if err != nil {
		bootstrapFail("resolving home directory", err)
	}

	ctx := context.Background()

	fmt.Println("Provisioning Postgres...")
	databaseURL, err := provisionPostgres(ctx, home)
	if err != nil {
		bootstrapFail("provisioning Postgres", err)
	}
	// provisionPostgres returns a host-perspective URL (127.0.0.1) — every
	// container we run below (migration, loopback app, and eventually the
	// real Quadlet-managed unit) lives in its own network namespace, not the
	// host's, so it has to reach Postgres via Podman's host-mapped DNS name
	// instead. Using the exact same hostname here as the real deploy will
	// later use (rather than 127.0.0.1 now and something else after
	// handoff) means the envVar row the bootstrap API stores never needs
	// special-casing between the two.
	containerDatabaseURL := strings.Replace(databaseURL, "127.0.0.1", "host.containers.internal", 1)

	fmt.Println("Provisioning the private container registry...")
	registryCreds, err := provisionRegistry(ctx, home, *domain)
	if err != nil {
		bootstrapFail("provisioning the registry", err)
	}
	fmt.Printf("Registry ready at 127.0.0.1:%d (public host: %s once Caddy is configured for it).\n", registryPort, registryCreds.Host)

	resolvedImage := *image
	if resolvedImage == "" {
		fmt.Println()
		fmt.Println("No --image given: building the control-plane image from source instead")
		fmt.Println("(there's no build pipeline running yet to do this the normal way — that's")
		fmt.Println("what this bootstrap is standing up). This can take several minutes.")
		built, err := buildAndPushSelfImage(ctx, home, *source, registryCreds)
		if err != nil {
			bootstrapFail("building the control-plane image", err)
		}
		resolvedImage = built
	}

	betterAuthSecret, err := randomBase64(32)
	if err != nil {
		bootstrapFail("generating BETTER_AUTH_SECRET", err)
	}
	encryptionKey, err := randomBase64(32)
	if err != nil {
		bootstrapFail("generating ENCRYPTION_KEY", err)
	}
	bootstrapSecret, err := randomHex(24)
	if err != nil {
		bootstrapFail("generating bootstrap secret", err)
	}

	origin := "https://" + *domain
	containerEnv := map[string]string{
		"DATABASE_URL":                  containerDatabaseURL,
		"ORIGIN":                        origin,
		"BETTER_AUTH_SECRET":            betterAuthSecret,
		"ENCRYPTION_KEY":                encryptionKey,
		"BAKERY_BOOTSTRAP_SECRET":       bootstrapSecret,
		"BAKERY_SELF_IMAGE":             resolvedImage,
		"BAKERY_REGISTRY_HOST":          registryCreds.Host,
		"BAKERY_REGISTRY_PUBLIC_DOMAIN": registryCreds.Host,
		"BAKERY_REGISTRY_PUSH_USERNAME": registryCreds.PushUsername,
		"BAKERY_REGISTRY_PUSH_PASSWORD": registryCreds.PushPassword,
		"BAKERY_REGISTRY_PULL_USERNAME": registryCreds.PullUsername,
		"BAKERY_REGISTRY_PULL_PASSWORD": registryCreds.PullPassword,
		"GITHUB_CLIENT_ID":              githubClientID,
		"GITHUB_CLIENT_SECRET":          githubClientSecret,
		"GITHUB_APP_ID":                 githubAppID,
		"GITHUB_APP_SLUG":               githubAppSlug,
		"GITHUB_APP_PRIVATE_KEY":        githubAppPrivateKey,
		"GITHUB_WEBHOOK_SECRET":         githubWebhookSecret,
	}

	fmt.Println("Provisioning the build worker...")
	buildWorkerEnv := map[string]string{
		"DATABASE_URL":                  databaseURL, // host-perspective — this runs natively, not in a container
		"ENCRYPTION_KEY":                encryptionKey,
		"BAKERY_REGISTRY_HOST":          registryCreds.Host,
		"BAKERY_REGISTRY_PUSH_USERNAME": registryCreds.PushUsername,
		"BAKERY_REGISTRY_PUSH_PASSWORD": registryCreds.PushPassword,
		"GITHUB_APP_ID":                 githubAppID,
		"GITHUB_APP_PRIVATE_KEY":        githubAppPrivateKey,
	}
	if err := provisionBuildWorker(ctx, home, *source, buildWorkerEnv); err != nil {
		bootstrapFail("provisioning the build worker", err)
	}

	fmt.Printf("Running database migrations (%s)...\n", resolvedImage)
	if err := runMigrations(ctx, resolvedImage, containerEnv); err != nil {
		bootstrapFail("running migrations", err)
	}

	fmt.Println("Starting the control plane, loopback-only...")
	if err := startLoopbackContainer(ctx, resolvedImage, containerEnv); err != nil {
		bootstrapFail("starting the control plane", err)
	}

	loopbackURL := fmt.Sprintf("http://127.0.0.1:%d", loopbackPort)
	fmt.Println("Waiting for it to become healthy...")
	if err := waitForHTTP(ctx, loopbackURL+"/login", 2*time.Minute); err != nil {
		bootstrapFail("waiting for the control plane to start", err)
	}

	fmt.Println("Creating the first admin account, guild, host, and app...")
	apiRequest := bootstrapAPIRequest{
		Email:                *adminEmail,
		Password:             *adminPassword,
		GuildName:            *guildName,
		Domain:               *domain,
		DatabaseURL:          containerDatabaseURL,
		Origin:               origin,
		BetterAuthSecret:     betterAuthSecret,
		EncryptionKey:        encryptionKey,
		RegistryHost:         registryCreds.Host,
		RegistryPushUsername: registryCreds.PushUsername,
		RegistryPushPassword: registryCreds.PushPassword,
		RegistryPullUsername: registryCreds.PullUsername,
		RegistryPullPassword: registryCreds.PullPassword,
		GithubClientID:       githubClientID,
		GithubClientSecret:   githubClientSecret,
		GithubAppID:          githubAppID,
		GithubAppSlug:        githubAppSlug,
		GithubAppPrivateKey:  githubAppPrivateKey,
		GithubWebhookSecret:  githubWebhookSecret,
	}
	hostToken, err := callBootstrapAPI(ctx, loopbackURL, apiRequest, bootstrapSecret)
	if err != nil {
		if strings.Contains(err.Error(), "already completed") {
			// A previous run got far enough to create the user/guild/host but
			// was interrupted before finishing the join handoff below — the
			// self-disabling gate now (correctly) refuses to run again, but
			// this run has no host token to hand off with. Postgres and the
			// loopback container are already fine either way; nothing here
			// needs to be undone.
			bootstrapFail("calling the bootstrap API",
				fmt.Errorf("%w — a previous run already created the admin/guild/host, but this run has no host token; find the host's token in the Bakery UI (or reissue one) and run 'bakery join --token=... --url=%s' directly", err, origin))
		}
		bootstrapFail("calling the bootstrap API", err)
	}

	// Points the daemon at the *loopback* URL first, not the real public
	// origin — found live: the real domain has nothing serving it yet at
	// this point (Caddy's Caddyfile is still just the empty skeleton, no
	// site block, no TLS cert for it), so a daemon pointed there from the
	// start can never successfully check in at all, which means it can
	// never receive the very "deploy" command that would be the thing to
	// fix that — a real chicken-and-egg circularity, not a transient
	// timing issue. `host.last_seen_at` stayed NULL and the `deploy`
	// hostCommand stayed `pending` indefinitely against the real domain,
	// confirmed against an actual box, with Caddy and the daemon both
	// genuinely running the whole time. The loopback control plane is
	// already up and serving `/api/v1/agent/checkin` right now — that's
	// reachable immediately.
	fmt.Println("Enrolling this host (bakery join)...")
	if err := writeDaemonConfig(home, hostToken, loopbackURL); err != nil {
		bootstrapFail("writing daemon config", err)
	}
	if err := installDaemonUnit(home); err != nil {
		bootstrapFail("installing daemon systemd unit", err)
	}
	if err := setupCaddyQuadlet(home); err != nil {
		bootstrapFail("setting up Caddy", err)
	}
	bestEffortUnprivilegedPorts()

	fmt.Println("Waiting for the real deployment to come online (this can take a few minutes)...")
	if err := waitForHTTP(ctx, origin+"/login", 5*time.Minute); err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "WARNING: %s isn't responding yet: %v\n", origin, err)
		fmt.Fprintln(os.Stderr, "The daemon and host are enrolled — check 'systemctl --user status bakery-daemon caddy'")
		fmt.Fprintln(os.Stderr, "and the host's status in the Bakery UI once it's reachable.")
	} else {
		// Now that the real deployment has taken over and Caddy has a real
		// site block + certificate for it, switch the daemon to the public
		// URL for good — it can't keep pointing at loopback, since the
		// loopback container is about to be removed. `installDaemonUnit`'s
		// `enable --now` won't pick up the rewritten env file on its own
		// (systemd only reads EnvironmentFile= at process start), so this
		// needs an explicit restart, not just a re-enable.
		fmt.Println("Real deployment is healthy — switching the daemon to the public URL...")
		if err := writeDaemonConfig(home, hostToken, origin); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to switch the daemon to %s: %v\n", origin, err)
		} else if err := runCmd("systemctl", "--user", "restart", "bakery-daemon.service"); err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: failed to restart the daemon after switching its URL: %v\n", err)
		}
		fmt.Println("Stopping the temporary loopback instance...")
		_ = stopLoopbackContainer(ctx)
	}

	fmt.Println()
	fmt.Println("=== Bootstrap complete ===")
	fmt.Printf("URL:   %s\n", origin)
	fmt.Printf("Admin: %s\n", *adminEmail)
	fmt.Println("(Passwords and secrets were never printed — they're stored encrypted.)")
}

func bootstrapFail(step string, err error) {
	fmt.Fprintf(os.Stderr, "bakery: bootstrap: %s: %v\n", step, err)
	os.Exit(1)
}

// missingRequiredGithubEnvVars returns the sorted (for stable, testable
// output) names of any empty values — pulled out of cmdBootstrap so this
// specific check (found live: silently-optional here meant a real, crash-
// looping container two provisioning steps and several minutes later) has
// a unit test of its own, not just end-to-end coverage.
func missingRequiredGithubEnvVars(vars map[string]string) []string {
	var missing []string
	for name, value := range vars {
		if value == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func promptLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func randomBase64(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// checkDNSPointsHere ports the old root-bootstrap script's DNS precondition
// check into Go: `bakery join`'s Caddy can't get a real certificate for
// `domain` unless it already resolves to this exact box.
func checkDNSPointsHere(domain string) error {
	boxIP, err := publicIPv4()
	if err != nil {
		return fmt.Errorf("determining this box's public IPv4 address: %w", err)
	}

	ips, err := net.LookupHost(domain)
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("%s does not resolve to anything yet — point its DNS A record at %s and re-run", domain, boxIP)
	}

	var resolvedIP string
	for _, ip := range ips {
		if parsed := net.ParseIP(ip); parsed != nil && parsed.To4() != nil {
			resolvedIP = ip
			break
		}
	}
	if resolvedIP == "" {
		return fmt.Errorf("%s has no IPv4 A record (only: %s)", domain, strings.Join(ips, ", "))
	}
	if resolvedIP != boxIP {
		return fmt.Errorf("%s resolves to %s, but this box is %s — fix DNS before re-running", domain, resolvedIP, boxIP)
	}
	return nil
}

// publicIPv4 mirrors the old root-bootstrap script's `curl -4 ifconfig.me` — the
// dialer is forced to tcp4 so a dual-stack box's IPv6 address (which a bare
// `http.Get` might prefer) never gets compared against the domain's A
// record by mistake.
func publicIPv4() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "tcp4", addr)
			},
		},
	}
	// /ip specifically, not the root path: ifconfig.me's root only returns
	// plain text when it recognizes the client as a CLI tool (curl's
	// default User-Agent) — Go's own default User-Agent ("Go-http-client/…")
	// isn't recognized, so the root path serves the full HTML page instead,
	// which then got treated as "the IP" and always failed the comparison
	// below. /ip is a dedicated plain-text endpoint, unaffected by User-Agent.
	resp, err := client.Get("https://ifconfig.me/ip")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// writeEnvFile writes KEY=VALUE lines to a mode-0600 temp file — used for
// `podman run --env-file` instead of `-e KEY=value` args, so secrets never
// show up in `ps` output for the duration of the container's startup.
func writeEnvFile(env map[string]string) (string, error) {
	f, err := os.CreateTemp("", "bakery-bootstrap-env-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return "", err
	}
	for key, value := range env {
		if _, err := fmt.Fprintf(f, "%s=%s\n", key, value); err != nil {
			return "", err
		}
	}
	return f.Name(), nil
}

func runMigrations(ctx context.Context, image string, env map[string]string) error {
	envFile, err := writeEnvFile(env)
	if err != nil {
		return err
	}
	defer os.Remove(envFile)

	cmd := exec.CommandContext(ctx, "podman", "run", "--rm",
		"--env-file", envFile,
		image, "bun", "node_modules/.bin/drizzle-kit", "migrate",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// startLoopbackContainer starts the control-plane image with its HTTP port
// published to 127.0.0.1 only (`-p 127.0.0.1:3000:3000`) — never a public
// interface, regardless of what address the app itself binds to inside its
// own network namespace. `--replace` makes re-running bootstrap after an
// interrupted attempt safe rather than erroring on an already-existing
// container name.
func startLoopbackContainer(ctx context.Context, image string, env map[string]string) error {
	envFile, err := writeEnvFile(env)
	if err != nil {
		return err
	}
	defer os.Remove(envFile)

	cmd := exec.CommandContext(ctx, "podman", "run", "-d", "--replace",
		"--name", loopbackContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", loopbackPort, loopbackPort),
		"--env-file", envFile,
		image,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func stopLoopbackContainer(ctx context.Context) error {
	return exec.CommandContext(ctx, "podman", "rm", "-f", loopbackContainerName).Run()
}

func waitForHTTP(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
				lastErr = fmt.Errorf("status %d", resp.StatusCode)
			} else {
				lastErr = err
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("timed out after %s: %w", timeout, lastErr)
}

type bootstrapAPIRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	GuildName        string `json:"guildName"`
	Domain           string `json:"domain"`
	DatabaseURL      string `json:"databaseUrl"`
	Origin           string `json:"origin"`
	BetterAuthSecret string `json:"betterAuthSecret"`
	EncryptionKey    string `json:"encryptionKey"`

	// Persisted as real envVar rows on the self-app (Phase 08 task 14) so
	// the *real*, post-handoff deployment has them too, not just the
	// temporary loopback container this same process is about to start —
	// without this, the "Connect GitHub" step (task 13) and every future
	// build would have nothing to authenticate with once the loopback
	// container is torn down.
	RegistryHost         string `json:"registryHost"`
	RegistryPushUsername string `json:"registryPushUsername"`
	RegistryPushPassword string `json:"registryPushPassword"`
	RegistryPullUsername string `json:"registryPullUsername"`
	RegistryPullPassword string `json:"registryPullPassword"`

	// Required, not optional: SvelteKit's `defineEnvVars` (src/env.ts)
	// declares all six of these as needed for the app to boot at all, so
	// `cmdBootstrap` already fails fast before any of this runs if any are
	// missing from its own environment — found live, the hard way, when
	// "optional" here just meant a silently crash-looping container later.
	GithubClientID      string `json:"githubClientId"`
	GithubClientSecret  string `json:"githubClientSecret"`
	GithubAppID         string `json:"githubAppId"`
	GithubAppSlug       string `json:"githubAppSlug"`
	GithubAppPrivateKey string `json:"githubAppPrivateKey"`
	GithubWebhookSecret string `json:"githubWebhookSecret"`
}

type bootstrapAPIResponse struct {
	HostToken string `json:"hostToken"`
}

func callBootstrapAPI(ctx context.Context, baseURL string, reqBody bootstrapAPIRequest, bootstrapSecret string) (string, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/bootstrap", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bootstrap-Secret", bootstrapSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bootstrap API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed bootstrapAPIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("decoding bootstrap API response: %w", err)
	}
	if parsed.HostToken == "" {
		return "", fmt.Errorf("bootstrap API response had no hostToken")
	}
	return parsed.HostToken, nil
}
