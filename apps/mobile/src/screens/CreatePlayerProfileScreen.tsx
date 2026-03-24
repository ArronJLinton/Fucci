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
  Alert,
} from 'react-native';
import {useNavigation} from '@react-navigation/native';
import type {NavigationProp} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {LinearGradient} from 'expo-linear-gradient';
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

const BG = '#030712';
const CARD = '#111827';
const BORDER = '#1f2937';
const MUTED = '#94a3b8';
const LABEL = '#9ca3af';
const LIME = '#c7f349';
const WHITE = '#e5e7eb';

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
        club: isFreeAgent ? null : club.trim() || null,
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

  const handleAddPhoto = () => {
    Alert.alert(
      'Add photo',
      'Profile photos will be available in a future update.',
    );
  };

  const flag = countryCode ? countryCodeToFlag(countryCode) : '';
  const positionLabel = position
    ? POSITIONS.find(p => p.value === position)?.label ?? position
    : '';

  return (
    <View style={styles.root}>
      <LinearGradient
        colors={['transparent', 'rgba(6,78,59,0.35)', 'rgba(15,118,110,0.25)']}
        style={styles.footerGlow}
        pointerEvents="none"
      />
      <SafeAreaView style={styles.safe} edges={['top']}>
        <KeyboardAvoidingView
          style={styles.keyboard}
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
          <View style={styles.header}>
            <TouchableOpacity
              onPress={() => navigation.goBack()}
              style={styles.backBtn}
              accessibilityLabel="Back">
              <Ionicons name="arrow-back" size={24} color={LIME} />
            </TouchableOpacity>
            <Text style={styles.headerTitle}>Player Profile</Text>
            <View style={styles.headerRight} />
          </View>

          <ScrollView
            style={styles.scroll}
            contentContainerStyle={styles.scrollContent}
            keyboardShouldPersistTaps="handled"
            showsVerticalScrollIndicator={false}>
            <View style={styles.photoBlock}>
              <View style={styles.photoDashed}>
                <Ionicons name="camera-outline" size={40} color={MUTED} />
                <View style={styles.photoPlus}>
                  <Ionicons name="add" size={14} color={BG} />
                </View>
              </View>
              <TouchableOpacity
                style={styles.addPhotoBtn}
                onPress={handleAddPhoto}
                activeOpacity={0.85}
                accessibilityLabel="Add photo">
                <Text style={styles.addPhotoBtnText}>ADD PHOTO</Text>
              </TouchableOpacity>
            </View>

            <Text style={styles.stepLabel}>Step 01 of 03</Text>

            <View style={styles.fieldBlock}>
              <Text style={styles.fieldLabel}>Date of birth</Text>
              <View style={styles.inputRow}>
                <Ionicons
                  name="calendar-outline"
                  size={20}
                  color={MUTED}
                  style={styles.rowIcon}
                />
                <TextInput
                  style={styles.inputFlex}
                  placeholder={`Age (${MIN_AGE}–${MAX_AGE})`}
                  placeholderTextColor={MUTED}
                  value={age}
                  onChangeText={setAge}
                  keyboardType="number-pad"
                  maxLength={2}
                  editable={!submitting}
                  accessibilityLabel="Age in years"
                />
              </View>
              <Text style={styles.fieldHint}>
                Enter your age in years (optional).
              </Text>
            </View>

            <TouchableOpacity
              style={styles.fieldBlock}
              onPress={() => setShowCountryPicker(true)}
              activeOpacity={0.85}
              accessibilityLabel="Select nationality"
              accessibilityRole="button">
              <Text style={styles.fieldLabel}>Nationality</Text>
              <View style={styles.inputRow}>
                <Ionicons
                  name="globe-outline"
                  size={20}
                  color={MUTED}
                  style={styles.rowIcon}
                />
                <Text
                  style={[
                    styles.selectText,
                    !countryName && styles.placeholderText,
                  ]}
                  numberOfLines={1}>
                  {flag ? `${flag} ` : ''}
                  {countryName || 'Select Country'}
                </Text>
                <Ionicons name="chevron-down" size={20} color={MUTED} />
              </View>
            </TouchableOpacity>

            <View style={styles.fieldBlock}>
              <View style={styles.clubLabelRow}>
                <Text style={styles.fieldLabel}>Current club</Text>
                <View style={styles.freeAgentInline}>
                  <Text style={styles.freeAgentCaps}>FREE AGENT</Text>
                  <Switch
                    value={isFreeAgent}
                    onValueChange={setIsFreeAgent}
                    trackColor={{false: '#374151', true: LIME}}
                    thumbColor={isFreeAgent ? '#030712' : '#9ca3af'}
                    ios_backgroundColor="#374151"
                    accessibilityLabel="Free agent toggle"
                  />
                </View>
              </View>
              <View style={styles.inputRow}>
                <Ionicons
                  name="football-outline"
                  size={20}
                  color={MUTED}
                  style={styles.rowIcon}
                />
                <TextInput
                  style={styles.inputFlex}
                  placeholder="Search for your club..."
                  placeholderTextColor={MUTED}
                  value={club}
                  onChangeText={setClub}
                  editable={!isFreeAgent && !submitting}
                  accessibilityLabel="Club name"
                />
              </View>
            </View>

            <TouchableOpacity
              style={styles.fieldBlock}
              onPress={() => setShowPositionPicker(true)}
              activeOpacity={0.85}
              accessibilityLabel="Select primary position"
              accessibilityRole="button">
              <Text style={styles.fieldLabel}>Primary position</Text>
              <View style={styles.inputRow}>
                <Ionicons
                  name="git-network-outline"
                  size={20}
                  color={MUTED}
                  style={styles.rowIcon}
                />
                <Text
                  style={[
                    styles.selectText,
                    !position && styles.placeholderText,
                  ]}
                  numberOfLines={1}>
                  {position ? positionLabel : 'Choose Position'}
                </Text>
                <Ionicons name="chevron-down" size={20} color={MUTED} />
              </View>
            </TouchableOpacity>

            {error ? <Text style={styles.error}>{error}</Text> : null}

            <TouchableOpacity
              style={[styles.nextBtn, submitting && styles.nextBtnDisabled]}
              onPress={handleNext}
              disabled={submitting}
              activeOpacity={0.9}
              accessibilityLabel="Next"
              accessibilityRole="button">
              {submitting ? (
                <ActivityIndicator color={BG} />
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
      </SafeAreaView>

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
          <TouchableOpacity
            style={styles.positionBackdrop}
            activeOpacity={1}
            onPress={() => setShowPositionPicker(false)}
          />
          <View style={styles.positionSheet}>
            <Text style={styles.positionSheetTitle}>Primary position</Text>
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
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: BG,
  },
  footerGlow: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    height: '42%',
  },
  safe: {
    flex: 1,
    backgroundColor: 'transparent',
  },
  keyboard: {
    flex: 1,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  backBtn: {
    padding: 8,
    minWidth: 44,
  },
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 13,
    fontWeight: '800',
    letterSpacing: 2,
    color: LIME,
    textTransform: 'uppercase',
  },
  headerRight: {
    width: 44,
  },
  scroll: {
    flex: 1,
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingBottom: 40,
  },
  photoBlock: {
    alignItems: 'center',
    marginTop: 8,
    marginBottom: 20,
  },
  photoDashed: {
    width: 132,
    height: 132,
    borderRadius: 8,
    borderWidth: 2,
    borderColor: '#4b5563',
    borderStyle: 'dashed',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: CARD,
  },
  photoPlus: {
    position: 'absolute',
    bottom: 10,
    right: 10,
    width: 22,
    height: 22,
    borderRadius: 11,
    backgroundColor: LIME,
    alignItems: 'center',
    justifyContent: 'center',
  },
  addPhotoBtn: {
    marginTop: -14,
    paddingHorizontal: 18,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: LIME,
  },
  addPhotoBtnText: {
    fontSize: 11,
    fontWeight: '800',
    color: BG,
    letterSpacing: 0.5,
  },
  stepLabel: {
    textAlign: 'center',
    fontSize: 11,
    fontWeight: '600',
    letterSpacing: 1.2,
    color: MUTED,
    textTransform: 'uppercase',
    marginBottom: 22,
  },
  fieldBlock: {
    marginBottom: 18,
  },
  fieldLabel: {
    fontSize: 10,
    fontWeight: '700',
    letterSpacing: 1,
    color: LABEL,
    textTransform: 'uppercase',
    marginBottom: 8,
  },
  fieldHint: {
    fontSize: 11,
    color: '#6b7280',
    marginTop: 6,
    marginLeft: 4,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: CARD,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: BORDER,
    paddingHorizontal: 14,
    paddingVertical: 14,
    minHeight: 52,
  },
  rowIcon: {
    marginRight: 12,
  },
  inputFlex: {
    flex: 1,
    fontSize: 15,
    color: WHITE,
    paddingVertical: 0,
  },
  selectText: {
    flex: 1,
    fontSize: 15,
    color: WHITE,
  },
  placeholderText: {
    color: MUTED,
  },
  clubLabelRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  freeAgentInline: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  freeAgentCaps: {
    fontSize: 9,
    fontWeight: '800',
    letterSpacing: 0.8,
    color: MUTED,
  },
  error: {
    color: '#f87171',
    fontSize: 14,
    marginBottom: 14,
    textAlign: 'center',
  },
  nextBtn: {
    backgroundColor: LIME,
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 12,
  },
  nextBtnDisabled: {
    opacity: 0.75,
  },
  nextBtnText: {
    color: BG,
    fontSize: 15,
    fontWeight: '800',
    letterSpacing: 1,
  },
  maybeLater: {
    alignItems: 'center',
    marginTop: 18,
    paddingVertical: 8,
  },
  maybeLaterText: {
    fontSize: 11,
    fontWeight: '600',
    letterSpacing: 1.2,
    color: MUTED,
    textTransform: 'uppercase',
  },
  positionOverlay: {
    ...StyleSheet.absoluteFillObject,
    justifyContent: 'flex-end',
  },
  positionBackdrop: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0,0,0,0.6)',
  },
  positionSheet: {
    backgroundColor: '#0f172a',
    borderTopLeftRadius: 18,
    borderTopRightRadius: 18,
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 36,
    borderTopWidth: 1,
    borderColor: BORDER,
  },
  positionSheetTitle: {
    fontSize: 16,
    fontWeight: '700',
    marginBottom: 12,
    color: WHITE,
  },
  positionOption: {
    paddingVertical: 16,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: BORDER,
  },
  positionOptionText: {
    fontSize: 16,
    color: WHITE,
  },
  positionCancel: {
    marginTop: 16,
    alignItems: 'center',
  },
  positionCancelText: {
    fontSize: 15,
    color: MUTED,
  },
});
