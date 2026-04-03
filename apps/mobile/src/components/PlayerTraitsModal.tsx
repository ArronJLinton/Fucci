import React, {useState, useEffect} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  TouchableOpacity,
  ScrollView,
  ActivityIndicator,
  Pressable,
} from 'react-native';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {PLAYER_TRAIT_CODES} from '../types/playerProfile';
import {PLAYER_TRAIT_LABELS, TraitHexImage} from './player_traits';

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
      } else {
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
      onRequestClose={onDismiss}
      presentationStyle="fullScreen">
      <SafeAreaView style={styles.safe} edges={['top', 'bottom']}>
        <View style={styles.container}>
          <View style={styles.header}>
            <View style={styles.headerSpacer} />
            <Text style={styles.title} numberOfLines={1}>
              Select traits
            </Text>
            <View style={styles.headerSide}>
              <Pressable
                onPress={onDismiss}
                style={({pressed}) => [
                  styles.headerBtn,
                  pressed && styles.headerBtnPressed,
                ]}
                hitSlop={12}
                accessibilityLabel="Close"
                accessibilityRole="button">
                <Ionicons name="close" size={24} color="#f8fafc" />
              </Pressable>
            </View>
          </View>
          <Text style={styles.hint}>
            Tap traits to select or deselect them. Save when you are done.
          </Text>
        <ScrollView
          style={styles.list}
          contentContainerStyle={styles.listContent}
          keyboardShouldPersistTaps="handled">
          {PLAYER_TRAIT_CODES.map(code => {
            const isSelected = selected.has(code);
            return (
              <TouchableOpacity
                key={code}
                style={styles.row}
                onPress={() => toggle(code)}
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
      </SafeAreaView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  safe: {
    flex: 1,
    backgroundColor: '#2c2c2c',
  },
  container: {
    flex: 1,
    backgroundColor: '#2c2c2c',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: '#404040',
  },
  headerSpacer: {
    width: 48,
  },
  headerSide: {
    width: 48,
    alignItems: 'flex-end',
    justifyContent: 'center',
  },
  headerBtn: {
    padding: 8,
    minWidth: 40,
    minHeight: 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerBtnPressed: {
    opacity: 0.7,
  },
  title: {
    flex: 1,
    fontSize: 17,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
    textAlign: 'center',
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
