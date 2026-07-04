import { GUILDS, type Guild } from '$lib/data/bakery';

class GuildStore {
	guilds = $state<Record<string, Guild>>({ ...GUILDS });

	get list() {
		return Object.values(this.guilds);
	}

	add(guild: Guild) {
		this.guilds[guild.id] = guild;
		// mutate original so non-reactive pages that import GUILDS directly can find it
		GUILDS[guild.id] = guild;
	}

	findByInvite(code: string) {
		return this.list.find((g) => g.invite.toLowerCase() === code.toLowerCase()) ?? null;
	}
}

export const guildStore = new GuildStore();
