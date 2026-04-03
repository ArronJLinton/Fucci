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
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {COMPARE_PLAYERS_CATALOG} from '../data/comparePlayersCatalog';
import type {ComparePlayerSnapshot} from '../types/comparePlayer';

type Props = {
  visible: boolean;
  onClose: () => void;
  onSelect: (player: ComparePlayerSnapshot) => void;
  /** Hide the player already shown on the left column */
  excludeIds?: Set<string>;
};

export function ComparePlayerSearchModal({
  visible,
  onClose,
  onSelect,
  excludeIds,
}: Props) {
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return COMPARE_PLAYERS_CATALOG.filter(p => {
      if (excludeIds?.has(p.id)) return false;
      if (!q) return true;
      return (
        p.displayName.toLowerCase().includes(q) ||
        p.team.toLowerCase().includes(q) ||
        p.countryLabel.toLowerCase().includes(q)
      );
    });
  }, [query, excludeIds]);

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

          <FlatList
            data={filtered}
            keyExtractor={item => item.id}
            keyboardShouldPersistTaps="handled"
            contentContainerStyle={styles.listContent}
            ListEmptyComponent={
              <Text style={styles.empty}>No players match your search.</Text>
            }
            renderItem={({item}) => (
              <TouchableOpacity
                style={styles.row}
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
                <Ionicons name="chevron-forward" size={20} color="#475569" />
              </TouchableOpacity>
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
  listContent: {paddingBottom: 32},
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1e293b',
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
