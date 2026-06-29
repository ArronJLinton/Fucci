import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  Switch,
  ActivityIndicator,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import {usePushPreferences} from '../hooks/usePushPreferences';

const LIME = '#c7f349';
const CARD = '#0b1224';
const CARD_BORDER = '#1f2937';
const MUTED = '#64748b';
const TEXT = '#e2e8f0';

type Props = {
  /** When true, show section header + hint (Settings). When false, inline card only (Account). */
  showHeader?: boolean;
  compact?: boolean;
};

export default function PushNotificationSettings({
  showHeader = true,
  compact = false,
}: Props) {
  const {
    prefs,
    loading,
    saving,
    isLoggedIn,
    handleMasterToggle,
  } = usePushPreferences();

  return (
    <View style={compact ? styles.compactWrap : undefined}>
      {showHeader ? (
        <>
          <View style={styles.sectionHeaderRow}>
            <Ionicons name="notifications" size={16} color={LIME} />
            <Text style={styles.sectionHeaderText}>NOTIFICATIONS</Text>
          </View>
          <Text style={styles.sectionHint}>
            {isLoggedIn
              ? 'Get World Cup updates on debates, news, and matches'
              : 'Sign in to enable push notifications'}
          </Text>
        </>
      ) : null}

      <View style={styles.card}>
        {loading ? (
          <View style={styles.prefsLoading}>
            <ActivityIndicator color={LIME} />
          </View>
        ) : (
          <>
            <ToggleRow
              label="Enable push notifications"
              value={prefs.master_enabled}
              onValueChange={handleMasterToggle}
              disabled={saving || !isLoggedIn}
              last
            />
          </>
        )}
      </View>
    </View>
  );
}

function ToggleRow({
  label,
  value,
  onValueChange,
  last,
  disabled,
}: {
  label: string;
  value: boolean;
  onValueChange: (v: boolean) => void;
  last?: boolean;
  disabled?: boolean;
}) {
  return (
    <View style={[styles.toggleRow, !last && styles.rowBorder]}>
      <Text style={[styles.toggleLabel, disabled && styles.toggleLabelMuted]}>
        {label}
      </Text>
      <Switch
        value={value}
        onValueChange={onValueChange}
        disabled={disabled}
        trackColor={{false: '#334155', true: 'rgba(199,243,73,0.45)'}}
        thumbColor={value ? LIME : '#94a3b8'}
        ios_backgroundColor="#334155"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  compactWrap: {
    marginBottom: 16,
  },
  sectionHeaderRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 10,
    marginTop: 8,
  },
  sectionHeaderText: {
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.6,
    color: LIME,
  },
  sectionHint: {
    fontSize: 12,
    color: MUTED,
    marginBottom: 10,
    marginTop: -4,
  },
  card: {
    backgroundColor: CARD,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: CARD_BORDER,
    paddingHorizontal: 14,
    paddingVertical: 4,
    marginBottom: 20,
  },
  prefsLoading: {
    paddingVertical: 24,
    alignItems: 'center',
  },
  rowBorder: {
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: 'rgba(148,163,184,0.15)',
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
  },
  toggleLabel: {
    fontSize: 12,
    fontWeight: '800',
    letterSpacing: 1.2,
    color: TEXT,
  },
  toggleLabelMuted: {
    color: MUTED,
  },
});
