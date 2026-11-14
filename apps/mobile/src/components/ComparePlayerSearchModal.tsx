import React, {useMemo, useState} from 'react';
import {
  View,
  Text,
  Modal,
  StyleSheet,
  TextInput,
  TouchableOpacity,
  FlatList,
  KeyboardAvoidingView,
  Platform,
  Alert,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import type {ComparePlayerSnapshot} from '../types/comparePlayer';
import {useAuth} from '../context/AuthContext';
import {promptBlockUser, promptReportContent} from '../services/moderation';

type Props = {
  visible: boolean;
  players: ComparePlayerSnapshot[];
  loading?: boolean;
  loadError?: string | null;
  onClose: () => void;
  onSelect: (player: ComparePlayerSnapshot) => void;
  /** Hide the player already shown on the left column */
  excludeIds?: Set<string>;
};

export function ComparePlayerSearchModal({
  visible,
  players,
  loading,
  loadError,
  onClose,
  onSelect,
  excludeIds,
}: Props) {
  const {token} = useAuth();
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return players.filter(p => {
      if (excludeIds?.has(p.id)) return false;
      if (!q) return true;
      return (
        p.displayName.toLowerCase().includes(q) ||
        p.team.toLowerCase().includes(q) ||
        p.countryLabel.toLowerCase().includes(q)
      );
    });
  }, [players, query, excludeIds]);

  const onReportPlayer = (item: ComparePlayerSnapshot) => {
    if (!token || item.userId == null) {
      Alert.alert(
        'Unavailable',
        'Sign in to report or block this player profile.',
      );
      return;
    }
    promptReportContent({
      token,
      reportableType: 'player_profile',
      reportableId: String(item.userId),
      title: 'Report player profile',
    });
  };

  const onBlockPlayer = (item: ComparePlayerSnapshot) => {
    if (!token || item.userId == null) {
      Alert.alert(
        'Unavailable',
        'Sign in to report or block this player profile.',
      );
      return;
    }
    promptBlockUser({
      token,
      blockedUserId: item.userId,
      reportableType: 'player_profile',
      reportableId: String(item.userId),
    });
  };

  return (
    <Modal
      visible={visible}
      animationType="slide"
      presentationStyle="pageSheet"
      onRequestClose={onClose}>
      <SafeAreaView style={styles.safe} edges={['top']}>
        <KeyboardAvoidingView
          style={styles.flex}
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
          <View style={styles.header}>
            <TouchableOpacity
              onPress={onClose}
              style={styles.headerBtn}
              accessibilityLabel="Close">
              <Ionicons name="close" size={26} color="#e2e8f0" />
            </TouchableOpacity>
            <Text style={styles.headerTitle}>Select player</Text>
            <View style={styles.headerBtn} />
          </View>

          <View style={styles.searchRow}>
            <Ionicons name="search" size={20} color="#64748b" />
            <TextInput
              style={styles.searchInput}
              placeholder="Search name, club, country…"
              placeholderTextColor="#64748b"
              value={query}
              onChangeText={setQuery}
              autoCorrect={false}
              autoCapitalize="none"
              clearButtonMode="while-editing"
            />
          </View>
          <Text style={styles.hint}>
            Use Report or Block on a player row, or tap the row to select them.
          </Text>

          <FlatList
            data={filtered}
            keyExtractor={item => item.id}
            keyboardShouldPersistTaps="handled"
            contentContainerStyle={styles.listContent}
            ListEmptyComponent={
              <Text style={styles.empty}>
                {loading
                  ? 'Loading players...'
                  : loadError
                    ? loadError
                    : 'No players match your search.'}
              </Text>
            }
            renderItem={({item}) => (
              <View style={styles.row}>
                <TouchableOpacity
                  style={styles.rowMain}
                  onPress={() => {
                    onSelect(item);
                    setQuery('');
                    onClose();
                  }}
                  activeOpacity={0.7}>
                  <View style={styles.rowAvatar}>
                    <Ionicons name="person" size={22} color="#64748b" />
                  </View>
                  <View style={styles.rowText}>
                    <Text style={styles.rowName}>{item.displayName}</Text>
                    <Text style={styles.rowMeta}>
                      {item.team} · {item.positionAbbrev}
                    </Text>
                  </View>
                </TouchableOpacity>
                <TouchableOpacity
                  style={styles.moderationBtn}
                  onPress={() => onReportPlayer(item)}
                  accessibilityRole="button"
                  accessibilityLabel="Report player profile">
                  <Ionicons name="flag-outline" size={18} color="#94a3b8" />
                </TouchableOpacity>
                <TouchableOpacity
                  style={styles.moderationBtn}
                  onPress={() => onBlockPlayer(item)}
                  accessibilityRole="button"
                  accessibilityLabel="Block user">
                  <Ionicons name="hand-left-outline" size={18} color="#94a3b8" />
                </TouchableOpacity>
              </View>
            )}
          />
        </KeyboardAvoidingView>
      </SafeAreaView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  flex: {flex: 1},
  safe: {flex: 1, backgroundColor: '#0a0e17'},
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1e293b',
  },
  headerBtn: {width: 44, height: 44, alignItems: 'center', justifyContent: 'center'},
  headerTitle: {
    fontSize: 16,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.5,
  },
  searchRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginHorizontal: 16,
    marginTop: 12,
    marginBottom: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    borderRadius: 12,
    backgroundColor: '#111827',
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  searchInput: {
    flex: 1,
    fontSize: 16,
    color: '#e2e8f0',
    padding: 0,
  },
  hint: {
    marginHorizontal: 16,
    marginBottom: 8,
    fontSize: 12,
    color: '#64748b',
  },
  listContent: {paddingBottom: 32},
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1e293b',
    gap: 4,
  },
  rowMain: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    minWidth: 0,
  },
  moderationBtn: {
    width: 40,
    height: 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  rowAvatar: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#1e293b',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
  },
  rowText: {flex: 1, minWidth: 0},
  rowName: {
    fontSize: 15,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.3,
  },
  rowMeta: {marginTop: 4, fontSize: 12, color: '#94a3b8', fontWeight: '600'},
  empty: {
    textAlign: 'center',
    color: '#64748b',
    marginTop: 40,
    paddingHorizontal: 24,
    fontSize: 14,
  },
});
