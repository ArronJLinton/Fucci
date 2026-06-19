/// <reference types="jest" />

import type {DebateSummary} from '../../types/debate';
import {
  debatePassesActivityFilters,
  debatePassesDiscoveryFilters,
} from '../debateFilters';

const NOW_MS = Date.parse('2026-06-17T12:00:00Z');

function debate(overrides: Partial<DebateSummary> = {}): DebateSummary {
  return {
    id: 1,
    match_id: '12345',
    headline: 'Should Bellingham start against Senegal?',
    description: 'A tactical debate before kickoff.',
    debate_type: 'pre_match',
    ai_generated: true,
    created_at: '2026-06-17T10:00:00Z',
    updated_at: '2026-06-17T10:00:00Z',
    ...overrides,
  };
}

describe('debate feed filters', () => {
  it('keeps recent voted activity even when discovery World Cup keywords miss', () => {
    const recentVoted = debate();

    expect(debatePassesDiscoveryFilters(recentVoted, NOW_MS, true)).toBe(false);
    expect(debatePassesActivityFilters(recentVoted, NOW_MS)).toBe(true);
  });

  it('still filters stale voted activity by generation age', () => {
    const oldVoted = debate({created_at: '2026-06-10T10:00:00Z'});

    expect(debatePassesActivityFilters(oldVoted, NOW_MS)).toBe(false);
  });

  it('keeps World Cup keyword matches in discovery lists', () => {
    const worldCupDebate = debate({
      source_headline: 'FIFA World Cup lineup questions continue',
    });

    expect(debatePassesDiscoveryFilters(worldCupDebate, NOW_MS, true)).toBe(
      true,
    );
  });
});
