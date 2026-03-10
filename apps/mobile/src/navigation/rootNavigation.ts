import {createNavigationContainerRef} from '@react-navigation/native';
import type {RootStackParamList} from '../types/navigation';

export const rootNavigationRef =
  createNavigationContainerRef<RootStackParamList>();

export function rootNavigate(
  name: keyof RootStackParamList,
  params?: RootStackParamList[keyof RootStackParamList],
) {
  if (rootNavigationRef.isReady()) {
    rootNavigationRef.navigate(name as any, params as any);
  }
}
