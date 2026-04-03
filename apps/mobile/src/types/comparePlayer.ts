/** Snapshot used to render one column on the compare screen (signed-in user or catalog pick). */
export type ComparePlayerSnapshot = {
  id: string;
  displayName: string;
  age: number | null;
  countryCode: string;
  /** Uppercase display, e.g. BRAZIL */
  countryLabel: string;
  team: string;
  positionAbbrev: string;
  photoUrl: string | null;
  /** Overall-style rating for head-to-head cards */
  rating: number;
  speed: number;
  shooting: number;
  passing: number;
  dribbling: number;
  defending: number;
  physical: number;
  stamina: number;
  /** e.g. €150M or — */
  valueLabel: string;
  seasonGoals: number;
  seasonLabel: string;
};
