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

const MAX_TRAITS = 5;

const TRAIT_LABELS: Record<string, string> = {
  LEADERSHIP: 'Leadership',
  FINESSE_SHOT: 'Finesse Shot',
  PLAYMAKER: 'Playmaker',
  SPEED_DRIBBLER: 'Speed Dribbler',
  LONG_SHOT_TAKER: 'Long Shot Taker',
  OUTSIDE_FOOT_SHOT: 'Outside Foot Shot',
  POWER_HEADER: 'Power Header',
  FLAIR: 'Flair',
  POWER_FREE_KICK: 'Power Free Kick',
};

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
            <Ionicons name="arrow-back" size={24} color="#000" />
          </TouchableOpacity>
          <Text style={styles.title}>Select Player Traits</Text>
          <TouchableOpacity
            onPress={onDismiss}
            style={styles.headerBtn}
            accessibilityLabel="Close">
            <Ionicons name="close" size={24} color="#000" />
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
                accessibilityLabel={`${TRAIT_LABELS[code] || code}, ${isSelected ? 'selected' : 'not selected'}`}
                accessibilityRole="checkbox"
                accessibilityState={{checked: isSelected}}>
                <View style={styles.rowIcon}>
                  <Ionicons
                    name="ellipse-outline"
                    size={22}
                    color={isSelected ? '#22c55e' : '#ccc'}
                  />
                </View>
                <Text style={styles.rowLabel}>{TRAIT_LABELS[code] || code}</Text>
                {isSelected ? (
                  <Ionicons name="checkmark-circle" size={24} color="#22c55e" />
                ) : null}
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
    backgroundColor: '#fff',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  headerBtn: {
    padding: 8,
    minWidth: 40,
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000',
  },
  hint: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    fontSize: 14,
    color: '#6b7280',
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
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#eee',
  },
  rowDisabled: {
    opacity: 0.6,
  },
  rowIcon: {
    marginRight: 12,
  },
  rowLabel: {
    flex: 1,
    fontSize: 16,
    color: '#000',
  },
  footer: {
    padding: 20,
    paddingBottom: 32,
    borderTopWidth: 1,
    borderTopColor: '#eee',
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
