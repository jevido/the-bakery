import { createCipheriv, createDecipheriv, randomBytes } from 'node:crypto';

const ALGORITHM = 'aes-256-gcm';
const IV_LENGTH = 12;
const AUTH_TAG_LENGTH = 16;

/**
 * Key rotation: generate a new ENCRYPTION_KEY, decrypt every envVar row with
 * the old key and re-encrypt with the new one in a single migration pass
 * before swapping the env var — there's no dual-key support, so old and new
 * ciphertexts can't coexist mid-rotation.
 */
function encryptionKey(): Buffer {
	const encoded = process.env.ENCRYPTION_KEY;
	if (!encoded) throw new Error('ENCRYPTION_KEY is not set');
	const key = Buffer.from(encoded, 'base64');
	if (key.length !== 32) throw new Error('ENCRYPTION_KEY must decode to exactly 32 bytes');
	return key;
}

/** AES-256-GCM encrypt with a random IV per call. Output: base64(iv):base64(ciphertext+authTag). */
export function encrypt(plaintext: string): string {
	const iv = randomBytes(IV_LENGTH);
	const cipher = createCipheriv(ALGORITHM, encryptionKey(), iv);
	const ciphertext = Buffer.concat([cipher.update(plaintext, 'utf-8'), cipher.final()]);
	const authTag = cipher.getAuthTag();
	return `${iv.toString('base64')}:${Buffer.concat([ciphertext, authTag]).toString('base64')}`;
}

/** Reverses `encrypt`. Throws if the ciphertext is malformed or has been tampered with. */
export function decrypt(ciphertext: string): string {
	const [ivPart, dataPart] = ciphertext.split(':');
	if (!ivPart || !dataPart) throw new Error('Malformed ciphertext');

	const iv = Buffer.from(ivPart, 'base64');
	const data = Buffer.from(dataPart, 'base64');
	if (data.length < AUTH_TAG_LENGTH) throw new Error('Malformed ciphertext');

	const authTag = data.subarray(data.length - AUTH_TAG_LENGTH);
	const encrypted = data.subarray(0, data.length - AUTH_TAG_LENGTH);

	const decipher = createDecipheriv(ALGORITHM, encryptionKey(), iv);
	decipher.setAuthTag(authTag);
	const plaintext = Buffer.concat([decipher.update(encrypted), decipher.final()]);
	return plaintext.toString('utf-8');
}
