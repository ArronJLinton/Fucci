import React, {createContext, useContext} from 'react';
import type {ScrollHandlerProcessed} from 'react-native-reanimated';

export type MatchDetailsScrollContextValue = {
  scrollHandler: ScrollHandlerProcessed<Record<string, unknown>>;
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
