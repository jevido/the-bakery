<script lang="ts">
	import { enhance } from '$app/forms';
	import type { ActionData } from './$types';

	let { form }: { form: ActionData } = $props();
	let submitting = $state(false);
</script>

<svelte:head><title>Log in — The Bakery</title></svelte:head>

<div
	class="flex h-screen w-screen items-center justify-center bg-[var(--main)] text-[var(--tx)] font-bakery"
>
	<div
		class="w-full max-w-[380px] p-6 rounded-[14px] bg-[var(--panel)] border border-[var(--line)]"
	>
		<h1 class="text-[18px] font-heading font-bold text-[var(--tx)] mb-1">Welcome back</h1>
		<p class="text-[13px] text-[var(--tx-3)] mb-6">Log in to your Bakery account.</p>

		<form
			method="post"
			action="?/signIn"
			use:enhance={() => {
				submitting = true;
				return async ({ update }) => {
					await update();
					submitting = false;
				};
			}}
			class="space-y-4"
		>
			<label class="block">
				<span class="text-[12px] text-[var(--tx-2)] mb-1.5 block">Email</span>
				<input
					type="email"
					name="email"
					required
					autocomplete="email"
					class="w-full h-9 px-3 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] placeholder:text-[var(--tx-3)] outline-none focus:border-[var(--line-2)] transition-colors"
				/>
			</label>
			<label class="block">
				<span class="text-[12px] text-[var(--tx-2)] mb-1.5 block">Password</span>
				<input
					type="password"
					name="password"
					required
					autocomplete="current-password"
					class="w-full h-9 px-3 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] placeholder:text-[var(--tx-3)] outline-none focus:border-[var(--line-2)] transition-colors"
				/>
			</label>

			{#if form?.message}
				<p class="text-[12px] text-[#f0836b]">{form.message}</p>
			{/if}

			<button
				type="submit"
				disabled={submitting}
				class="w-full h-9 rounded-[8px] bg-[var(--grn)] text-[#07130c] text-[13px] font-semibold hover:bg-[var(--grn-2)] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
			>
				{submitting ? 'Logging in…' : 'Log in'}
			</button>
		</form>

		<div class="flex items-center gap-3 my-5">
			<div class="h-px flex-1 bg-[var(--line)]"></div>
			<span class="text-[11px] text-[var(--tx-3)]">or</span>
			<div class="h-px flex-1 bg-[var(--line)]"></div>
		</div>

		<form method="post" action="?/signInSocial" use:enhance>
			<button
				type="submit"
				class="w-full h-9 rounded-[8px] bg-[var(--card-2)] border border-[var(--line)] text-[var(--tx)] text-[13px] font-semibold hover:border-[var(--line-2)] hover:bg-white/[0.04] transition-colors"
			>
				Continue with GitHub
			</button>
		</form>

		<p class="text-[12px] text-[var(--tx-3)] text-center mt-6">
			Don't have an account? <a href="/signup" class="text-[var(--tx)] underline">Sign up</a>
		</p>
	</div>
</div>
