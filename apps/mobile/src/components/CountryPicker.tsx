import React, {useState, useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  TouchableOpacity,
  TextInput,
  FlatList,
  ListRenderItem,
} from 'react-native';
import {Ionicons} from '@expo/vector-icons';
import {COUNTRIES, countryCodeToFlag} from '../data/countries';

export interface CountryPickerProps {
  visible: boolean;
  onDismiss: () => void;
  onSelect: (countryCode: string, displayName: string) => void;
  selectedCode?: string | null;
  showFlags?: boolean;
}

export function CountryPicker({
  visible,
  onDismiss,
  onSelect,
  selectedCode = null,
  showFlags = true,
}: CountryPickerProps) {
  const [search, setSearch] = useState('');

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return COUNTRIES;
    return COUNTRIES.filter(
      c =>
        c.name.toLowerCase().includes(q) || c.code.toLowerCase().includes(q),
    );
  }, [search]);

  const handleSelect = (code: string, name: string) => {
    onSelect(code, name);
    onDismiss();
  };

  const renderItem: ListRenderItem<{code: string; name: string}> = ({
    item,
  }) => {
    const flag = showFlags ? countryCodeToFlag(item.code) : '';
    const isSelected = selectedCode === item.code;
    return (
      <TouchableOpacity
        style={[styles.row, isSelected && styles.rowSelected]}
        onPress={() => handleSelect(item.code, item.name)}
        activeOpacity={0.7}
        accessibilityLabel={`${item.name}, ${item.code}`}
        accessibilityRole="button">
        {flag ? <Text style={styles.flag}>{flag}</Text> : null}
        <Text style={styles.name} numberOfLines={1}>
          {item.name}
        </Text>
        <Ionicons
          name="chevron-forward"
          size={18}
          color="#999"
          style={styles.chevron}
        />
      </TouchableOpacity>
    );
  };

  return (
    <Modal
      visible={visible}
      transparent
      animationType="slide"
      onRequestClose={onDismiss}>
      <View style={styles.container}>
        <View style={styles.sheet}>
          <View style={styles.header}>
            <TouchableOpacity
              onPress={onDismiss}
              style={styles.closeBtn}
              accessibilityLabel="Close"
              accessibilityRole="button">
              <Ionicons name="arrow-back" size={24} color="#000" />
            </TouchableOpacity>
            <Text style={styles.title}>Select Country</Text>
            <TouchableOpacity
              onPress={onDismiss}
              style={styles.closeBtn}
              accessibilityLabel="Close"
              accessibilityRole="button">
              <Ionicons name="close" size={24} color="#000" />
            </TouchableOpacity>
          </View>
          <TextInput
            style={styles.search}
            placeholder="Search country..."
            placeholderTextColor="#999"
            value={search}
            onChangeText={setSearch}
            autoCapitalize="none"
            autoCorrect={false}
            accessibilityLabel="Search country"
          />
          <FlatList
            data={filtered}
            keyExtractor={item => item.code}
            renderItem={renderItem}
            style={styles.list}
            keyboardShouldPersistTaps="handled"
            ListEmptyComponent={
              <Text style={styles.empty}>No countries match your search.</Text>
            }
          />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'flex-end',
  },
  sheet: {
    backgroundColor: '#fff',
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    maxHeight: '80%',
    paddingBottom: 24,
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
  closeBtn: {
    padding: 8,
    minWidth: 40,
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000',
  },
  search: {
    marginHorizontal: 16,
    marginVertical: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
    backgroundColor: '#f5f5f5',
    borderRadius: 10,
    fontSize: 16,
    color: '#000',
  },
  list: {
    maxHeight: 400,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#eee',
  },
  rowSelected: {
    backgroundColor: '#f0f9ff',
  },
  flag: {
    fontSize: 22,
    marginRight: 12,
  },
  name: {
    flex: 1,
    fontSize: 16,
    color: '#000',
  },
  chevron: {
    marginLeft: 8,
  },
  empty: {
    padding: 24,
    textAlign: 'center',
    color: '#666',
    fontSize: 15,
  },
});
