import React, {useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  TextInput,
  ScrollView,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Switch,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import type {NavigationProp} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {CountryPicker} from '../components/CountryPicker';
import {countryCodeToFlag} from '../data/countries';
import {createPlayerProfile} from '../services/playerProfile';
import type {PlayerProfileInput, PlayerPosition} from '../types/playerProfile';

const POSITIONS: {value: PlayerPosition; label: string}[] = [
  {value: 'GK', label: 'Goalkeeper'},
  {value: 'DEF', label: 'Defender'},
  {value: 'MID', label: 'Midfielder'},
  {value: 'FWD', label: 'Forward'},
];

const MIN_AGE = 13;
const MAX_AGE = 60;

export default function CreatePlayerProfileScreen() {
  const navigation = useNavigation<NavigationProp>();
  const {token} = useAuth();
  const [age, setAge] = useState('');
  const [countryCode, setCountryCode] = useState<string | null>(null);
  const [countryName, setCountryName] = useState<string>('');
  const [club, setClub] = useState('');
  const [isFreeAgent, setIsFreeAgent] = useState(false);
  const [position, setPosition] = useState<PlayerPosition | null>(null);
  const [showCountryPicker, setShowCountryPicker] = useState(false);
  const [showPositionPicker, setShowPositionPicker] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleNext = async () => {
    if (!token) {
      setError('You must be signed in to create a profile.');
      return;
    }
    if (!countryCode || !countryName.trim()) {
      setError('Please select a country.');
      return;
    }
    if (!position) {
      setError('Please select a position.');
      return;
    }
    const ageNum = age.trim() ? parseInt(age.trim(), 10) : null;
    if (ageNum !== null && (ageNum < MIN_AGE || ageNum > MAX_AGE)) {
      setError(`Age must be between ${MIN_AGE} and ${MAX_AGE}.`);
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const body: PlayerProfileInput = {
        country: countryCode,
        position,
        age: ageNum,
        club: isFreeAgent ? null : (club.trim() || null),
        is_free_agent: isFreeAgent,
      };
      const profile = await createPlayerProfile(token, body);
      if (profile) {
        navigation.replace('PlayerProfile');
      } else {
        setError('Failed to create profile. Please try again.');
      }
    } catch {
      setError('Something went wrong. Please try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleMaybeLater = () => {
    if (navigation.canGoBack()) {
      navigation.goBack();
    } else {
      navigation.navigate('Main');
    }
  };

  const flag = countryCode ? countryCodeToFlag(countryCode) : '';
  const positionLabel = position
    ? POSITIONS.find(p => p.value === position)?.label ?? position
    : '';

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <KeyboardAvoidingView
        style={styles.keyboard}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
        <View style={styles.header}>
          <TouchableOpacity
            onPress={() => navigation.goBack()}
            style={styles.backBtn}
            accessibilityLabel="Back">
            <Ionicons name="arrow-back" size={24} color="#000" />
          </TouchableOpacity>
          <Text style={styles.title}>Create Player Profile</Text>
          <View style={styles.headerRight} />
        </View>

        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}>
          {/* Placeholder for photo - spec shows Upload Photo; Phase 5 will add it */}
          <View style={styles.photoPlaceholder}>
            <Ionicons name="person-circle-outline" size={80} color="#ccc" />
            <Text style={styles.photoHint}>Photo (add later)</Text>
          </View>

          <View style={styles.field}>
            <Text style={styles.label}>Age</Text>
            <TextInput
              style={styles.input}
              placeholder={`${MIN_AGE}–${MAX_AGE}`}
              placeholderTextColor="#999"
              value={age}
              onChangeText={setAge}
              keyboardType="number-pad"
              maxLength={2}
              editable={!submitting}
              accessibilityLabel="Age"
            />
            <Ionicons
              name="calendar-outline"
              size={20}
              color="#999"
              style={styles.inputIcon}
            />
          </View>

          <TouchableOpacity
            style={styles.field}
            onPress={() => setShowCountryPicker(true)}
            accessibilityLabel="Select country"
            accessibilityRole="button">
            <Text style={styles.label}>Country</Text>
            <View style={styles.selectRow}>
              {flag ? <Text style={styles.flag}>{flag}</Text> : null}
              <Text
                style={[styles.selectValue, !countryName && styles.placeholder]}>
                {countryName || 'Select country'}
              </Text>
              <Ionicons name="chevron-forward" size={20} color="#999" />
            </View>
          </TouchableOpacity>

          <View style={styles.field}>
            <Text style={styles.label}>Club or Free Agent</Text>
            <View style={styles.freeAgentRow}>
              <TextInput
                style={[styles.input, isFreeAgent && styles.inputDisabled]}
                placeholder="Club name"
                placeholderTextColor="#999"
                value={club}
                onChangeText={setClub}
                editable={!isFreeAgent && !submitting}
                accessibilityLabel="Club name"
              />
              <View style={styles.toggleRow}>
                <Text style={styles.toggleLabel}>Free Agent</Text>
                <Switch
                  value={isFreeAgent}
                  onValueChange={setIsFreeAgent}
                  trackColor={{false: '#ddd', true: '#22c55e'}}
                  thumbColor="#fff"
                  accessibilityLabel="Free agent toggle"
                />
              </View>
            </View>
          </View>

          <TouchableOpacity
            style={styles.field}
            onPress={() => setShowPositionPicker(true)}
            accessibilityLabel="Select position"
            accessibilityRole="button">
            <Text style={styles.label}>Position</Text>
            <View style={styles.selectRow}>
              <Text
                style={[styles.selectValue, !position && styles.placeholder]}>
                {position ? `+ ${positionLabel}` : 'Select position'}
              </Text>
              <Ionicons name="chevron-forward" size={20} color="#999" />
            </View>
          </TouchableOpacity>

          {error ? <Text style={styles.error}>{error}</Text> : null}

          <TouchableOpacity
            style={[styles.nextBtn, submitting && styles.nextBtnDisabled]}
            onPress={handleNext}
            disabled={submitting}
            activeOpacity={0.8}
            accessibilityLabel="Next"
            accessibilityRole="button">
            {submitting ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.nextBtnText}>NEXT</Text>
            )}
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.maybeLater}
            onPress={handleMaybeLater}
            disabled={submitting}
            accessibilityLabel="Maybe later">
            <Text style={styles.maybeLaterText}>Or Maybe Later</Text>
          </TouchableOpacity>
        </ScrollView>
      </KeyboardAvoidingView>

      <CountryPicker
        visible={showCountryPicker}
        onDismiss={() => setShowCountryPicker(false)}
        onSelect={(code, name) => {
          setCountryCode(code);
          setCountryName(name);
        }}
        selectedCode={countryCode}
      />

      {showPositionPicker ? (
        <View style={styles.positionOverlay}>
          <View style={styles.positionSheet}>
            <Text style={styles.positionSheetTitle}>Position</Text>
            {POSITIONS.map(p => (
              <TouchableOpacity
                key={p.value}
                style={styles.positionOption}
                onPress={() => {
                  setPosition(p.value);
                  setShowPositionPicker(false);
                }}>
                <Text style={styles.positionOptionText}>{p.label}</Text>
              </TouchableOpacity>
            ))}
            <TouchableOpacity
              style={styles.positionCancel}
              onPress={() => setShowPositionPicker(false)}>
              <Text style={styles.positionCancelText}>Cancel</Text>
            </TouchableOpacity>
          </View>
        </View>
      ) : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  keyboard: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  backBtn: {
    padding: 4,
  },
  title: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000',
  },
  headerRight: {
    width: 32,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    padding: 24,
    paddingBottom: 48,
  },
  photoPlaceholder: {
    alignItems: 'center',
    marginBottom: 24,
  },
  photoHint: {
    marginTop: 8,
    fontSize: 14,
    color: '#999',
  },
  field: {
    marginBottom: 20,
  },
  label: {
    fontSize: 14,
    fontWeight: '500',
    color: '#374151',
    marginBottom: 8,
  },
  input: {
    borderWidth: 1,
    borderColor: '#e5e7eb',
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 12,
    paddingRight: 40,
    fontSize: 16,
    color: '#000',
    backgroundColor: '#fff',
  },
  inputDisabled: {
    backgroundColor: '#f5f5f5',
    color: '#999',
  },
  inputIcon: {
    position: 'absolute',
    right: 12,
    top: 42,
  },
  selectRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#e5e7eb',
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 14,
    backgroundColor: '#fff',
  },
  flag: {
    fontSize: 20,
    marginRight: 10,
  },
  selectValue: {
    flex: 1,
    fontSize: 16,
    color: '#000',
  },
  placeholder: {
    color: '#999',
  },
  freeAgentRow: {
    gap: 12,
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  toggleLabel: {
    fontSize: 15,
    color: '#374151',
  },
  error: {
    color: '#dc2626',
    fontSize: 14,
    marginBottom: 16,
  },
  nextBtn: {
    backgroundColor: '#22c55e',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  nextBtnDisabled: {
    opacity: 0.7,
  },
  nextBtnText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  maybeLater: {
    alignItems: 'center',
    marginTop: 16,
  },
  maybeLaterText: {
    fontSize: 15,
    color: '#6b7280',
  },
  positionOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'flex-end',
  },
  positionSheet: {
    backgroundColor: '#fff',
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    padding: 24,
    paddingBottom: 32,
  },
  positionSheetTitle: {
    fontSize: 18,
    fontWeight: '600',
    marginBottom: 16,
    color: '#000',
  },
  positionOption: {
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#eee',
  },
  positionOptionText: {
    fontSize: 16,
    color: '#000',
  },
  positionCancel: {
    marginTop: 16,
    alignItems: 'center',
  },
  positionCancelText: {
    fontSize: 16,
    color: '#6b7280',
  },
});
