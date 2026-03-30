import React, {createContext, useContext} from 'react';
import type {ScrollHandlerProcessed} from 'react-native-reanimated';

/** Pass from `MatchDetailsScreen` into tab screens so scroll-driven hero collapse always works (navigator-safe). */
export type MatchDetailsScrollHandler =
  ScrollHandlerProcessed<Record<string, unknown>>;

export type MatchDetailsScrollContextValue = {
  scrollHandler: MatchDetailsScrollHandler;
};

const MatchDetailsScrollContext =
  createContext<MatchDetailsScrollContextValue | null>(null);

export function MatchDetailsScrollProvider({
  children,
  value,
}: {
  children: React.ReactNode;
  value: MatchDetailsScrollContextValue;
}) {
  return (
    <MatchDetailsScrollContext.Provider value={value}>
      {children}
    </MatchDetailsScrollContext.Provider>
  );
}

export function useMatchDetailsScroll(): MatchDetailsScrollContextValue | null {
  return useContext(MatchDetailsScrollContext);
}
