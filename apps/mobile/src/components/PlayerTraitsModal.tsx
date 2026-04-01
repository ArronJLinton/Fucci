import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  TouchableOpacity,
  ScrollView,
  ActivityIndicator,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import {PLAYER_TRAIT_CODES} from '../types/playerProfile';
import {PLAYER_TRAIT_LABELS, TraitHexImage} from './player_traits';

const MAX_TRAITS = 5;

export interface PlayerTraitsModalProps {
  visible: boolean;
  onDismiss: () => void;
  selectedTraits: string[];
  onSave: (traits: string[]) => Promise<void> | void;
}

export function PlayerTraitsModal({
  visible,
  onDismiss,
  selectedTraits,
  onSave,
}: PlayerTraitsModalProps) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (visible) {
      setSelected(new Set(selectedTraits));
    }
  }, [visible, selectedTraits]);

  const toggle = (code: string) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(code)) {
        next.delete(code);
      } else if (next.size < MAX_TRAITS) {
        next.add(code);
      }
      return next;
    });
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(Array.from(selected));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      visible={visible}
      animationType="slide"
      onRequestClose={onDismiss}>
      <View style={styles.container}>
        <View style={styles.header}>
          <TouchableOpacity
            onPress={onDismiss}
            style={styles.headerBtn}
            accessibilityLabel="Back">
            <Ionicons name="arrow-back" size={24} color="#f8fafc" />
          </TouchableOpacity>
          <Text style={styles.title}>Select traits</Text>
          <TouchableOpacity
            onPress={onDismiss}
            style={styles.headerBtn}
            accessibilityLabel="Close">
            <Ionicons name="close" size={24} color="#f8fafc" />
          </TouchableOpacity>
        </View>
        <Text style={styles.hint}>Choose up to {MAX_TRAITS} traits.</Text>
        <ScrollView
          style={styles.list}
          contentContainerStyle={styles.listContent}
          keyboardShouldPersistTaps="handled">
          {PLAYER_TRAIT_CODES.map(code => {
            const isSelected = selected.has(code);
            const atMax = selected.size >= MAX_TRAITS && !isSelected;
            return (
              <TouchableOpacity
                key={code}
                style={[styles.row, atMax && styles.rowDisabled]}
                onPress={() => !atMax && toggle(code)}
                disabled={atMax}
                activeOpacity={0.7}
                accessibilityLabel={`${PLAYER_TRAIT_LABELS[code] || code}, ${isSelected ? 'selected' : 'not selected'}`}
                accessibilityRole="checkbox"
                accessibilityState={{checked: isSelected}}>
                <View style={styles.rowLeading}>
                  <TraitHexImage code={code} size={46} />
                  <Text style={styles.rowLabel}>
                    {(PLAYER_TRAIT_LABELS[code] || code).toUpperCase()}
                  </Text>
                </View>
                {isSelected ? (
                  <Ionicons name="checkmark-circle" size={24} color="#4ade80" />
                ) : (
                  <Ionicons name="ellipse-outline" size={24} color="#64748b" />
                )}
              </TouchableOpacity>
            );
          })}
        </ScrollView>
        <View style={styles.footer}>
          <TouchableOpacity
            style={[styles.saveBtn, saving && styles.saveBtnDisabled]}
            onPress={handleSave}
            disabled={saving}
            accessibilityLabel="Save traits">
            {saving ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <Text style={styles.saveBtnText}>SAVE</Text>
            )}
          </TouchableOpacity>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#2c2c2c',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#404040',
  },
  headerBtn: {
    padding: 8,
    minWidth: 40,
  },
  title: {
    fontSize: 17,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
  },
  hint: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    fontSize: 14,
    color: '#94a3b8',
  },
  list: {
    flex: 1,
  },
  listContent: {
    padding: 16,
    paddingBottom: 24,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    gap: 12,
    paddingVertical: 12,
    paddingHorizontal: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#404040',
  },
  rowDisabled: {
    opacity: 0.6,
  },
  rowLeading: {
    flex: 1,
    minWidth: 0,
    flexDirection: 'column',
    alignItems: 'center',
    gap: 8,
  },
  rowTrailing: {
    paddingTop: 11,
  },
  rowLabel: {
    fontSize: 13,
    fontWeight: '800',
    color: '#4ade80',
    letterSpacing: 0.3,
    textAlign: 'center',
    alignSelf: 'stretch',
  },
  footer: {
    padding: 20,
    paddingBottom: 32,
    borderTopWidth: 1,
    borderTopColor: '#404040',
    backgroundColor: '#262626',
  },
  saveBtn: {
    backgroundColor: '#22c55e',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
  },
  saveBtnDisabled: {
    opacity: 0.7,
  },
  saveBtnText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
