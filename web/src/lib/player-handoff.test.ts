import { describe, expect, it } from 'vitest';
import { consumeContinuePlaying, markContinuePlaying } from './player-handoff';

describe('player-handoff', () => {
	it('solo se consume una vez', () => {
		expect(consumeContinuePlaying()).toBe(false);
		markContinuePlaying();
		expect(consumeContinuePlaying()).toBe(true);
		expect(consumeContinuePlaying()).toBe(false);
	});
});
