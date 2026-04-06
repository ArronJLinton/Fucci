import {
  useCallback,
  useEffect,
  useRef,
  type Dispatch,
  type SetStateAction,
} from 'react';

const CORE_MIN = 40;
const CORE_MAX = 99;

/** Delay before repeating; interval between repeats while held (ms). */
const HOLD_DELAY_MS = 400;
const REPEAT_MS = 70;

type SetNumber = Dispatch<SetStateAction<number>>;

/**
 * Press once to step; press and hold to repeat (after a short delay) until release.
 * Uses functional updates so repeat ticks always see the latest value.
 */
export function useHoldCoreStep(
  delta: number,
  setValue: SetNumber,
  disabled: boolean,
) {
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const clear = useCallback(() => {
    if (timeoutRef.current != null) {
      clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }
    if (intervalRef.current != null) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  const tick = useCallback(() => {
    setValue((s) => Math.max(CORE_MIN, Math.min(CORE_MAX, s + delta)));
  }, [delta, setValue]);

  const onPressIn = useCallback(() => {
    // Guard against duplicated press-in events so only one hold loop runs.
    clear();
    if (disabled) {
      return;
    }
    tick();
    timeoutRef.current = setTimeout(() => {
      intervalRef.current = setInterval(tick, REPEAT_MS);
    }, HOLD_DELAY_MS);
  }, [clear, disabled, tick]);

  const onPressOut = useCallback(() => {
    clear();
  }, [clear]);

  useEffect(() => {
    if (disabled) {
      clear();
    }
  }, [disabled, clear]);

  useEffect(() => () => clear(), [clear]);

  return {onPressIn, onPressOut};
}
