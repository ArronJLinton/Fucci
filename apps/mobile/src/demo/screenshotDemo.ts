/**
 * App Store screenshot fixtures — flip `APP_STORE_SCREENSHOT_MODE` to true locally.
 * Keep false in committed code; do not ship enabled builds.
 *
 * Matchday + debate team names, logos, and scores come from the staging API.
 */
import type {
  DebateComment,
  DebateFeedResponse,
  DebateResponse,
  DebateSummary,
  DebateTeams,
} from '../types/debate';
import type {Match} from '../types/match';
import {WORLD_CUP_LEAGUE_ID} from '../constants/leagues';
import {fetchMatchesForLocalDate} from '../services/futbol';
import {fetchDebatesByMatch} from '../services/debate';
import {makeApiRequest} from '../services/api';

export const APP_STORE_SCREENSHOT_MODE = false;

/** Local calendar day for Argentina 3-0 Algeria (kickoff 2026-06-17T01:00Z). */
export const SCREENSHOT_MATCHDAY = new Date(2026, 5, 16);

function isScreenshotMatchday(localDate: Date): boolean {
  return (
    localDate.getFullYear() === SCREENSHOT_MATCHDAY.getFullYear() &&
    localDate.getMonth() === SCREENSHOT_MATCHDAY.getMonth() &&
    localDate.getDate() === SCREENSHOT_MATCHDAY.getDate()
  );
}

export const SCREENSHOT_ARGENTINA_FIXTURE_ID = 1489381;
export const SCREENSHOT_FRANCE_FIXTURE_ID = 1489383;
export const SCREENSHOT_MBAPPE_DEBATE_ID = 9101;

const SCREENSHOT_DEBATE_CREATED = '2026-06-16T22:45:00Z';

/** Core "today" slate plus three more real WC group-stage fixtures for full-screen shots. */
const SCREENSHOT_TODAY_FIXTURE_IDS = [
  1489383, // France 3-1 Senegal
  1539016, // Iraq 1-4 Norway
  1489381, // Argentina 3-0 Algeria
  1489382, // Austria 3-1 Jordan
  1489384, // England 4-2 Croatia
  1539003, // Portugal 1-1 Congo DR
] as const;

async function fetchMatchesByUtcDate(
  dateStr: string,
): Promise<Match[]> {
  const data = await makeApiRequest(
    `/futbol/matches?date=${dateStr}&league_id=${WORLD_CUP_LEAGUE_ID}&season=2026`,
    'GET',
  );
  return data.response ?? [];
}

async function fetchScreenshotTodayMatches(): Promise<Match[]> {
  const batches = await Promise.all([
    fetchMatchesByUtcDate('2026-06-16'),
    fetchMatchesByUtcDate('2026-06-17'),
  ]);
  const byId = new Map<number, Match>();
  for (const batch of batches) {
    for (const match of batch) {
      if (match?.fixture?.id != null) {
        byId.set(match.fixture.id, match);
      }
    }
  }
  return SCREENSHOT_TODAY_FIXTURE_IDS.map(id => byId.get(id)).filter(
    (match): match is Match => match != null,
  );
}

/** Real matchday from GET /futbol/matches (World Cup, league 1). */
export async function fetchScreenshotMatchday(
  localDate: Date = SCREENSHOT_MATCHDAY,
): Promise<Match[]> {
  if (isScreenshotMatchday(localDate)) {
    const today = await fetchScreenshotTodayMatches();
    if (today.length >= 6) {
      return today;
    }
  }
  const rows = await fetchMatchesForLocalDate(
    localDate,
    WORLD_CUP_LEAGUE_ID,
    2026,
  );
  return rows ?? [];
}

type ApiDebateRow = {
  id: number;
  match_id: string;
  headline: string;
  description: string;
  debate_type: string;
  ai_generated: boolean;
  created_at: string;
  updated_at: string;
  teams?: DebateTeams;
  source_headline?: string;
};

function matchTeamsToDebateTeams(match: Match): DebateTeams {
  return {
    home: {
      name: match.teams.home.name,
      logo: match.teams.home.logo,
      score: match.goals.home ?? undefined,
    },
    away: {
      name: match.teams.away.name,
      logo: match.teams.away.logo,
      score: match.goals.away ?? undefined,
    },
  };
}

function toSummary(
  row: ApiDebateRow,
  consensus: {agree: number; disagree: number},
  extra?: Partial<DebateSummary>,
): DebateSummary {
  const total = consensus.agree + consensus.disagree;
  return {
    id: row.id,
    match_id: row.match_id,
    headline: row.headline,
    description: row.description,
    debate_type: row.debate_type,
    ai_generated: row.ai_generated,
    created_at: row.created_at,
    updated_at: row.updated_at,
    binary_consensus: {
      agree_upvotes: consensus.agree,
      disagree_upvotes: consensus.disagree,
    },
    teams: row.teams,
    source_headline: row.source_headline,
    analytics: {
      total_votes: total,
      total_comments: 3,
      engagement_score: total,
    },
    ...extra,
  };
}

async function firstPostMatchDebate(
  fixtureId: number,
): Promise<ApiDebateRow | null> {
  const rows = await fetchDebatesByMatch(fixtureId, 'post_match');
  const row = rows[0] as ApiDebateRow | undefined;
  return row ?? null;
}

/**
 * Debates feed for screenshots: hero uses real Argentina–Algeria fixture metadata;
 * voted rows are real post-match debates from the same API matchday slate.
 */
export async function fetchScreenshotDebatesFeed(): Promise<DebateFeedResponse> {
  const matches = await fetchScreenshotMatchday();
  const argMatch = matches.find(
    m => m.fixture.id === SCREENSHOT_ARGENTINA_FIXTURE_ID,
  );

  const heroTeams =
    argMatch != null
      ? matchTeamsToDebateTeams(argMatch)
      : undefined;
  const heroMatchDate = argMatch?.fixture.date;

  const hero: DebateSummary = {
    id: 9001,
    match_id: String(SCREENSHOT_ARGENTINA_FIXTURE_ID),
    headline:
      'Did Messi\'s hat-trick prove Argentina are the team to beat at this World Cup?',
    description:
      'Lionel Messi scored all three in a 3-0 win over Algeria on Matchday 1. Is this the start of another title run?',
    debate_type: 'post_match',
    ai_generated: true,
    created_at: argMatch?.fixture.date ?? new Date().toISOString(),
    updated_at: argMatch?.fixture.date ?? new Date().toISOString(),
    binary_consensus: {agree_upvotes: 847, disagree_upvotes: 312},
    teams: heroTeams,
    match_date: heroMatchDate,
    source_headline:
      'Messi hat-trick ties World Cup scoring record in Argentina win',
    analytics: {total_votes: 1159, total_comments: 3, engagement_score: 1159},
  };

  const votedFixtureIds = [1489382, 1539003, 1489384] as const;
  const votedConsensus = [
    {agree: 412, disagree: 588},
    {agree: 721, disagree: 279},
    {agree: 534, disagree: 466},
  ] as const;
  const votedOffsetsMs = [3600_000, 7200_000, 86400_000] as const;

  const votedRows = await Promise.all(
    votedFixtureIds.map(id => firstPostMatchDebate(id)),
  );

  const voted_debates: DebateSummary[] = votedRows
    .map((row, i) => {
      if (!row) {
        return null;
      }
      return toSummary(row, votedConsensus[i], {
        last_voted_at: new Date(Date.now() - votedOffsetsMs[i]).toISOString(),
      });
    })
    .filter((d): d is DebateSummary => d != null);

  return {new_debates: [hero], voted_debates};
}

function franceTeams(): DebateTeams {
  return {
    home: {
      name: 'France',
      logo: 'https://media.api-sports.io/football/teams/2.png',
      score: 3,
    },
    away: {
      name: 'Senegal',
      logo: 'https://media.api-sports.io/football/teams/13.png',
      score: 1,
    },
  };
}

/** Primary screenshot debate — Mbappé vs Dembélé pulse + thread. */
function buildMbappeDembeleDebate(): DebateResponse {
  return {
    id: SCREENSHOT_MBAPPE_DEBATE_ID,
    match_id: String(SCREENSHOT_FRANCE_FIXTURE_ID),
    debate_type: 'post_match',
    ai_generated: true,
    created_at: SCREENSHOT_DEBATE_CREATED,
    updated_at: SCREENSHOT_DEBATE_CREATED,
    headline:
      "Who is more impactful to France's attack, Mbappé or Dembélé?",
    description:
      "France's 3-1 win over Senegal had two different engines in the final third — direct runs in behind and creative width on the touchline.",
    teams: franceTeams(),
    cards: [
      {
        id: 910101,
        debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
        stance: 'agree',
        title: "Mbappé's runs in behind",
        description:
          'His pace stretched Senegal vertically and pulled defenders out of the block.',
        vote_counts: {upvotes: 632, downvotes: 74},
        user_vote: {vote_type: 'upvote'},
      },
      {
        id: 910102,
        debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
        stance: 'disagree',
        title: "Dembélé's creativity in wide areas",
        description:
          'His dribbling and switches of play opened the channels that led to big chances.',
        vote_counts: {upvotes: 406, downvotes: 0},
      },
    ],
  };
}

function buildFranceSupportDebates(): DebateResponse[] {
  return [
    {
      id: 9102,
      match_id: String(SCREENSHOT_FRANCE_FIXTURE_ID),
      debate_type: 'post_match',
      ai_generated: true,
      created_at: SCREENSHOT_DEBATE_CREATED,
      headline: "France's tactical approach was the difference maker",
      description:
        'Did Deschamps get the shape and pressing triggers right against a physical Senegal side?',
      teams: franceTeams(),
      cards: [
        {
          id: 910201,
          stance: 'agree',
          title: 'Tactics won the game',
          description: 'Structure and pressing created the chances.',
          vote_counts: {upvotes: 412, downvotes: 58},
        },
        {
          id: 910202,
          stance: 'disagree',
          title: 'Individual quality decided it',
          description: 'France won because of talent, not the game plan.',
          vote_counts: {upvotes: 289, downvotes: 0},
        },
      ],
    },
    {
      id: 9103,
      match_id: String(SCREENSHOT_FRANCE_FIXTURE_ID),
      debate_type: 'post_match',
      ai_generated: true,
      created_at: SCREENSHOT_DEBATE_CREATED,
      headline: "Senegal's World Cup exit is a massive disappointment",
      description:
        'Were the Lions of Teranga simply outclassed, or did they miss a window to push France?',
      teams: franceTeams(),
      cards: [
        {
          id: 910301,
          stance: 'agree',
          title: 'A missed opportunity',
          description: 'Senegal had spells to level it and could not convert.',
          vote_counts: {upvotes: 338, downvotes: 41},
        },
        {
          id: 910302,
          stance: 'disagree',
          title: 'France were just better',
          description: 'The gap in quality was clear over 90 minutes.',
          vote_counts: {upvotes: 271, downvotes: 0},
        },
      ],
    },
  ];
}

const SCREENSHOT_MATCH_DEBATES: DebateResponse[] = [
  buildMbappeDembeleDebate(),
  ...buildFranceSupportDebates(),
];

export function getScreenshotMatchDebates(match: Match): DebateResponse[] | null {
  if (!APP_STORE_SCREENSHOT_MODE) {
    return null;
  }
  if (match.fixture.id !== SCREENSHOT_FRANCE_FIXTURE_ID) {
    return null;
  }
  return SCREENSHOT_MATCH_DEBATES;
}

export function getScreenshotDebateById(
  debateId: number,
): DebateResponse | null {
  if (!APP_STORE_SCREENSHOT_MODE) {
    return null;
  }
  return SCREENSHOT_MATCH_DEBATES.find(d => d.id === debateId) ?? null;
}

export function getScreenshotDebateComments(
  debateId: number,
): DebateComment[] | null {
  if (!APP_STORE_SCREENSHOT_MODE || debateId !== SCREENSHOT_MBAPPE_DEBATE_ID) {
    return null;
  }

  return [
    {
      id: 9204,
      debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
      user_id: 103,
      user_display_name: 'NJ Alliance',
      content:
        "Dembélé's switch before the second goal broke Senegal's press. That's not luck — that's a winger reading the trap and punishing it.",
      created_at: '2026-06-16T23:24:00Z',
      net_score: 64,
      reactions: [{emoji: '👏', count: 9}],
    },
    {
      id: 9202,
      debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
      user_id: 101,
      user_display_name: 'LosBlancos',
      content:
        'Mbappé had three clear looks before half-time. If he buries one of those, this debate is not close — but he still forced the chaos that freed everyone else.',
      created_at: '2026-06-16T23:12:00Z',
      net_score: 51,
      reactions: [{emoji: '⚽', count: 6}],
    },
    {
      id: 9201,
      debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
      user_id: 1,
      user_display_name: 'Fucci',
      content:
        "Mbappé drew two defenders every time he touched it in the final third — but Dembélé's overlap is what stretched Senegal's back line. France need both profiles to go deep in this tournament.",
      created_at: '2026-06-16T23:05:00Z',
      net_score: 38,
      is_fucci_take: true,
      reactions: [{emoji: '🔥', count: 12}],
    },
    {
      id: 9203,
      debate_id: SCREENSHOT_MBAPPE_DEBATE_ID,
      user_id: 102,
      user_display_name: 'UnitedFC',
      content:
        "Deschamps built the central channel around Mbappé's runs. Dembélé flourished because that gravity opened the half-spaces — classic complementary partnership.",
      created_at: '2026-06-16T23:18:00Z',
      net_score: 19,
      reactions: [{emoji: '🧠', count: 3}],
    },
  ];
}
