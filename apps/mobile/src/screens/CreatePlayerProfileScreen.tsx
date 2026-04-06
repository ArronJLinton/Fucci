import React, {useEffect, useState} from 'react';
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
  Image,
  Modal,
  Pressable,
} from 'react-native';
import DateTimePicker from '@react-native-community/datetimepicker';
import {useNavigation} from '@react-navigation/native';
import type {NavigationProp} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {LinearGradient} from 'expo-linear-gradient';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {CountryPicker} from '../components/CountryPicker';
import {countryCodeToFlag} from '../data/countries';
import {userFacingApiMessage} from '../services/api';
import {
  createPlayerProfile,
  updatePlayerProfile,
  setPlayerProfileTraits,
} from '../services/playerProfile';
import {
  PLAYER_TRAIT_CODES,
  type PlayerProfile,
  type PlayerProfileInput,
  type PlayerPosition,
} from '../types/playerProfile';
import {PLAYER_TRAIT_LABELS, TraitHexImage} from '../components/player_traits';
import {
  DEFAULT_CORE_RATING,
  defaultCoreAttrs,
  defaultDribblingDefending,
} from '../utils/playerCoreAttrs';
import {useHoldCoreStep} from '../hooks/useHoldCoreStep';

const POSITIONS: {value: PlayerPosition; label: string}[] = [
  {value: 'GK', label: 'Goalkeeper'},
  {value: 'DEF', label: 'Defender'},
  {value: 'MID', label: 'Midfielder'},
  {value: 'FWD', label: 'Forward'},
];

const MIN_AGE = 13;
const MAX_AGE = 60;

/** Full years since birth; matches API age rules when DOB is in allowed range. */
function ageFromDateOfBirth(birth: Date): number {
  const today = new Date();
  let years = today.getFullYear() - birth.getFullYear();
  const m = today.getMonth() - birth.getMonth();
  if (m < 0 || (m === 0 && today.getDate() < birth.getDate())) {
    years--;
  }
  return years;
}

function minBirthDate(): Date {
  const d = new Date();
  d.setFullYear(d.getFullYear() - MAX_AGE);
  d.setHours(12, 0, 0, 0);
  return d;
}

function maxBirthDate(): Date {
  const d = new Date();
  d.setFullYear(d.getFullYear() - MIN_AGE);
  d.setHours(12, 0, 0, 0);
  return d;
}

function clampBirthDate(d: Date): Date {
  const t = d.getTime();
  const lo = minBirthDate().getTime();
  const hi = maxBirthDate().getTime();
  if (t < lo) {
    return new Date(lo);
  }
  if (t > hi) {
    return new Date(hi);
  }
  return d;
}

function defaultBirthDateForPicker(): Date {
  const d = new Date();
  d.setFullYear(d.getFullYear() - 25);
  d.setHours(12, 0, 0, 0);
  return clampBirthDate(d);
}

const BG = '#030712';
const CARD = '#111827';
const BORDER = '#1f2937';
const MUTED = '#94a3b8';
const LABEL = '#9ca3af';
const LIME = '#c7f349';
const WHITE = '#e5e7eb';

/** API / cached profile values may be missing or loosely typed; always yield 40–99. */
function clampCoreStat(raw: unknown, fallback: number): number {
  const fb = Math.max(40, Math.min(99, Math.round(fallback)));
  if (typeof raw === 'number' && Number.isFinite(raw)) {
    return Math.max(40, Math.min(99, Math.round(raw)));
  }
  if (typeof raw === 'string' && raw.trim() !== '') {
    const n = Number(raw);
    if (Number.isFinite(n)) {
      return Math.max(40, Math.min(99, Math.round(n)));
    }
  }
  return fb;
}

export default function CreatePlayerProfileScreen() {
  const navigation = useNavigation<NavigationProp>();
  const {token, user} = useAuth();
  const [wizardStep, setWizardStep] = useState<1 | 2 | 3>(1);
  const [dateOfBirth, setDateOfBirth] = useState<Date | null>(null);
  const [showDobPicker, setShowDobPicker] = useState(false);
  const [iosDobDraft, setIosDobDraft] = useState<Date>(() =>
    defaultBirthDateForPicker(),
  );
  const [countryCode, setCountryCode] = useState<string | null>(null);
  const [countryName, setCountryName] = useState<string>('');
  const [club, setClub] = useState('');
  const [isFreeAgent, setIsFreeAgent] = useState(false);
  const [position, setPosition] = useState<PlayerPosition | null>(null);
  const [showCountryPicker, setShowCountryPicker] = useState(false);
  const [showPositionPicker, setShowPositionPicker] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [selectedTraits, setSelectedTraits] = useState<string[]>([]);
  const baseCore = defaultCoreAttrs();
  const [editSpeed, setEditSpeed] = useState(baseCore.speed);
  const [editShooting, setEditShooting] = useState(baseCore.shooting);
  const [editPassing, setEditPassing] = useState(baseCore.passing);
  const [editPhysical, setEditPhysical] = useState(baseCore.physical);
  const [editStamina, setEditStamina] = useState(baseCore.stamina);

  const holdSpeedMinus = useHoldCoreStep(-1, setEditSpeed, submitting);
  const holdSpeedPlus = useHoldCoreStep(1, setEditSpeed, submitting);
  const holdShootingMinus = useHoldCoreStep(-1, setEditShooting, submitting);
  const holdShootingPlus = useHoldCoreStep(1, setEditShooting, submitting);
  const holdPassingMinus = useHoldCoreStep(-1, setEditPassing, submitting);
  const holdPassingPlus = useHoldCoreStep(1, setEditPassing, submitting);
  const holdPhysicalMinus = useHoldCoreStep(-1, setEditPhysical, submitting);
  const holdPhysicalPlus = useHoldCoreStep(1, setEditPhysical, submitting);
  const holdStaminaMinus = useHoldCoreStep(-1, setEditStamina, submitting);
  const holdStaminaPlus = useHoldCoreStep(1, setEditStamina, submitting);

  const applyCoreFromProfile = (p: PlayerProfile) => {
    const d = defaultCoreAttrs();
    setEditSpeed(clampCoreStat(p.speed, d.speed));
    setEditShooting(clampCoreStat(p.shooting, d.shooting));
    setEditPassing(clampCoreStat(p.passing, d.passing));
    setEditPhysical(clampCoreStat(p.physical, d.physical));
    setEditStamina(clampCoreStat(p.stamina, d.stamina));
  };

  /** If anything left the editors non-finite (stale API shape), fix when step 3 is shown. */
  useEffect(() => {
    if (wizardStep !== 3 || !position) {
      return;
    }
    const d = defaultCoreAttrs();
    setEditSpeed(s => clampCoreStat(s, d.speed));
    setEditShooting(s => clampCoreStat(s, d.shooting));
    setEditPassing(s => clampCoreStat(s, d.passing));
    setEditPhysical(s => clampCoreStat(s, d.physical));
    setEditStamina(s => clampCoreStat(s, d.stamina));
  }, [wizardStep, position]);

  const ageForApi = (): number | null => {
    if (!dateOfBirth) {
      return null;
    }
    return ageFromDateOfBirth(dateOfBirth);
  };

  const validateBasics = (): string | null => {
    if (!countryCode || !countryName.trim()) {
      return 'Please select a country.';
    }
    if (!position) {
      return 'Please select a position.';
    }
    const ageNum = ageForApi();
    if (ageNum !== null && (ageNum < MIN_AGE || ageNum > MAX_AGE)) {
      return `Age must be between ${MIN_AGE} and ${MAX_AGE}.`;
    }
    return null;
  };

  const buildBasicsBody = (): PlayerProfileInput | null => {
    if (!countryCode || !position) return null;
    const ageNum = ageForApi();
    return {
      country: countryCode,
      position,
      age: ageNum,
      club: isFreeAgent ? null : club.trim() || null,
      is_free_agent: isFreeAgent,
    };
  };

  const handleStep1Next = async () => {
    if (!token) {
      setError('You must be signed in to create a profile.');
      return;
    }
    const v = validateBasics();
    if (v) {
      setError(v);
      return;
    }
    const body = buildBasicsBody();
    if (!body) return;

    setError(null);
    setSubmitting(true);
    try {
      const p = await createPlayerProfile(token, body);
      applyCoreFromProfile(p);
      setWizardStep(2);
    } catch (err) {
      setError(userFacingApiMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleStep2Next = async () => {
    if (!token) {
      setError('You must be signed in.');
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      await setPlayerProfileTraits(token, selectedTraits);
      setWizardStep(3);
    } catch (err) {
      setError(userFacingApiMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleStep2Skip = () => {
    setError(null);
    setWizardStep(3);
  };

  const handleStep3Next = async () => {
    if (!token) {
      setError('You must be signed in.');
      return;
    }
    const v = validateBasics();
    if (v || !countryCode || !position) {
      setError(v ?? 'Missing profile data.');
      return;
    }
    const ageNum = ageForApi();
    const dd = defaultDribblingDefending();

    setError(null);
    setSubmitting(true);
    try {
      await updatePlayerProfile(token, {
        country: countryCode,
        position,
        age: ageNum,
        club: isFreeAgent ? null : club.trim() || null,
        is_free_agent: isFreeAgent,
        speed: editSpeed,
        shooting: editShooting,
        passing: editPassing,
        dribbling: dd.dribbling,
        defending: dd.defending,
        physical: editPhysical,
        stamina: editStamina,
      });
      navigation.replace('PlayerProfile');
    } catch (err) {
      setError(userFacingApiMessage(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleStep3Skip = () => {
    navigation.replace('PlayerProfile');
  };

  const handleMaybeLater = () => {
    if (navigation.canGoBack()) {
      navigation.goBack();
    } else {
      navigation.navigate('Main');
    }
  };

  const handleHeaderBack = () => {
    if (wizardStep > 1) {
      setWizardStep((s) => (s - 1) as 1 | 2 | 3);
      setError(null);
    } else if (navigation.canGoBack()) {
      navigation.goBack();
    } else {
      navigation.navigate('Main');
    }
  };

  const toggleTrait = (code: string) => {
    setSelectedTraits((prev) =>
      prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code],
    );
  };

  const defaultAvatarUri = user?.avatar_url?.trim() || null;

  const handleEditPhoto = () => {
    Alert.alert(
      'Profile photo',
      defaultAvatarUri
        ? 'Your account photo is shown by default. Player-specific photo upload will be available in a future update.'
        : 'Profile photo upload will be available in a future update. You can set an avatar in Settings to show it here.',
    );
  };

  const flag = countryCode ? countryCodeToFlag(countryCode) : '';
  const positionLabel = position
    ? POSITIONS.find((p) => p.value === position)?.label ?? position
    : '';

  const primaryAction =
    wizardStep === 1
      ? handleStep1Next
      : wizardStep === 2
        ? handleStep2Next
        : handleStep3Next;

  const showMaybeLater = wizardStep === 1;
  const showSkip = wizardStep === 2 || wizardStep === 3;
  const skipAction = wizardStep === 2 ? handleStep2Skip : handleStep3Skip;

  const dobDisplayText = dateOfBirth
    ? dateOfBirth.toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    : '';

  const openDobPicker = () => {
    setIosDobDraft(
      dateOfBirth ? clampBirthDate(dateOfBirth) : defaultBirthDateForPicker(),
    );
    setShowDobPicker(true);
  };

  const confirmIosDob = () => {
    setDateOfBirth(clampBirthDate(iosDobDraft));
    setShowDobPicker(false);
  };

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
              onPress={handleHeaderBack}
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
            {wizardStep === 1 ? (
              <View style={styles.photoBlock}>
                <TouchableOpacity
                  style={[
                    styles.photoDashed,
                    defaultAvatarUri && styles.photoDashedFilled,
                  ]}
                  onPress={handleEditPhoto}
                  activeOpacity={0.88}
                  accessibilityLabel="Edit profile photo"
                  accessibilityRole="button">
                  {defaultAvatarUri ? (
                    <Image
                      source={{uri: defaultAvatarUri}}
                      style={styles.photoAvatar}
                      resizeMode="cover"
                      accessibilityIgnoresInvertColors
                    />
                  ) : (
                    <Ionicons name="camera-outline" size={40} color={MUTED} />
                  )}
                  <View style={styles.photoEditBadge}>
                    <Ionicons name="pencil" size={12} color={BG} />
                  </View>
                </TouchableOpacity>
              </View>
            ) : null}

            <Text style={styles.stepLabel}>
              Step 0{wizardStep} of 03
            </Text>

            {wizardStep === 1 ? (
              <>
                <View style={styles.fieldBlock}>
                  <Text style={styles.fieldLabel}>Date of birth</Text>
                  <TouchableOpacity
                    style={styles.inputRow}
                    onPress={openDobPicker}
                    disabled={submitting}
                    activeOpacity={0.85}
                    accessibilityLabel="Date of birth"
                    accessibilityRole="button">
                    <Ionicons
                      name="calendar-outline"
                      size={20}
                      color={MUTED}
                      style={styles.rowIcon}
                    />
                    <Text
                      style={[
                        styles.selectText,
                        !dateOfBirth && styles.placeholderText,
                      ]}
                      numberOfLines={1}>
                      {dobDisplayText || 'Select date'}
                    </Text>
                    <Ionicons name="chevron-down" size={20} color={MUTED} />
                  </TouchableOpacity>
                  <View style={styles.dobHintRow}>
                    <Text style={styles.dobHintText}>
                      Optional. Used to set your age ({MIN_AGE}–{MAX_AGE}).
                    </Text>
                    {dateOfBirth ? (
                      <TouchableOpacity
                        onPress={() => setDateOfBirth(null)}
                        disabled={submitting}
                        hitSlop={{top: 8, bottom: 8, left: 8, right: 8}}
                        accessibilityLabel="Clear date of birth">
                        <Text style={styles.dobClear}>Clear</Text>
                      </TouchableOpacity>
                    ) : null}
                  </View>
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
                      placeholderTextColor={WHITE}
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
              </>
            ) : null}

            {wizardStep === 2 ? (
              <View style={styles.phaseSection}>
                <Text style={styles.phaseTitle}>Player traits</Text>
                <Text style={styles.phaseHint}>
                  Tap traits that fit your game. You can change these later on
                  your profile.
                </Text>
                {PLAYER_TRAIT_CODES.map((code) => {
                  const isSelected = selectedTraits.includes(code);
                  return (
                    <TouchableOpacity
                      key={code}
                      style={[
                        styles.traitRow,
                        isSelected && styles.traitRowSelected,
                      ]}
                      onPress={() => toggleTrait(code)}
                      activeOpacity={0.75}
                      accessibilityLabel={`${
                        PLAYER_TRAIT_LABELS[code] || code
                      }, ${isSelected ? 'selected' : 'not selected'}`}
                      accessibilityRole="checkbox"
                      accessibilityState={{checked: isSelected}}>
                      <View style={styles.traitRowLeading}>
                        <TraitHexImage code={code} size={46} />
                        <Text style={styles.traitRowLabel}>
                          {(PLAYER_TRAIT_LABELS[code] || code).toUpperCase()}
                        </Text>
                      </View>
                      {isSelected ? (
                        <Ionicons
                          name="checkmark-circle"
                          size={24}
                          color={LIME}
                        />
                      ) : (
                        <Ionicons
                          name="ellipse-outline"
                          size={24}
                          color="#64748b"
                        />
                      )}
                    </TouchableOpacity>
                  );
                })}
              </View>
            ) : null}

            {wizardStep === 3 ? (
              <View style={styles.phaseSection}>
                <Text style={styles.phaseTitle}>Core attributes</Text>
                <Text style={styles.phaseHint}>
                  Fine-tune your ratings. Dribbling and defending follow your
                  position.
                </Text>
                <View style={styles.coreCard}>
                  <Text style={styles.coreCardTitle}>Core attributes</Text>

                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Speed</Text>
                    <View style={styles.attrValueWrap}>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdSpeedMinus.onPressIn}
                        onPressOut={holdSpeedMinus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.attrValueText}>{editSpeed}</Text>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdSpeedPlus.onPressIn}
                        onPressOut={holdSpeedPlus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[styles.attrBarFill, {width: `${editSpeed}%`}]}
                    />
                  </View>

                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Shooting</Text>
                    <View style={styles.attrValueWrap}>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdShootingMinus.onPressIn}
                        onPressOut={holdShootingMinus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.attrValueText}>{editShooting}</Text>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdShootingPlus.onPressIn}
                        onPressOut={holdShootingPlus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${editShooting}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Passing</Text>
                    <View style={styles.attrValueWrap}>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdPassingMinus.onPressIn}
                        onPressOut={holdPassingMinus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.attrValueText}>{editPassing}</Text>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdPassingPlus.onPressIn}
                        onPressOut={holdPassingPlus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${editPassing}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Physical</Text>
                    <View style={styles.attrValueWrap}>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdPhysicalMinus.onPressIn}
                        onPressOut={holdPhysicalMinus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.attrValueText}>{editPhysical}</Text>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdPhysicalPlus.onPressIn}
                        onPressOut={holdPhysicalPlus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={[styles.attrBarTrack, styles.attrBarGapAfter]}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${editPhysical}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Stamina</Text>
                    <View style={styles.attrValueWrap}>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdStaminaMinus.onPressIn}
                        onPressOut={holdStaminaMinus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.attrValueText}>{editStamina}</Text>
                      <TouchableOpacity
                        style={styles.attrBtn}
                        onPressIn={holdStaminaPlus.onPressIn}
                        onPressOut={holdStaminaPlus.onPressOut}
                        disabled={submitting}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${editStamina}%`},
                      ]}
                    />
                  </View>
                </View>
              </View>
            ) : null}

            {error ? <Text style={styles.error}>{error}</Text> : null}

            <TouchableOpacity
              style={[styles.nextBtn, submitting && styles.nextBtnDisabled]}
              onPress={primaryAction}
              disabled={submitting}
              activeOpacity={0.9}
              accessibilityLabel="Next"
              accessibilityRole="button">
              {submitting ? (
                <ActivityIndicator color={BG} />
              ) : (
                <Text style={styles.nextBtnText}>
                  {wizardStep === 3 ? 'FINISH' : 'NEXT'}
                </Text>
              )}
            </TouchableOpacity>

            {showMaybeLater ? (
              <TouchableOpacity
                style={styles.maybeLater}
                onPress={handleMaybeLater}
                disabled={submitting}
                accessibilityLabel="Maybe later">
                <Text style={styles.maybeLaterText}>Or Maybe Later</Text>
              </TouchableOpacity>
            ) : null}

            {showSkip ? (
              <TouchableOpacity
                style={styles.maybeLater}
                onPress={skipAction}
                disabled={submitting}
                accessibilityLabel="Skip this step">
                <Text style={styles.maybeLaterText}>Skip</Text>
              </TouchableOpacity>
            ) : null}
          </ScrollView>
        </KeyboardAvoidingView>
      </SafeAreaView>

      {Platform.OS === 'android' && showDobPicker ? (
        <DateTimePicker
          value={dateOfBirth ?? defaultBirthDateForPicker()}
          mode="date"
          display="default"
          minimumDate={minBirthDate()}
          maximumDate={maxBirthDate()}
          onChange={(event, date) => {
            setShowDobPicker(false);
            if (event.type === 'dismissed') {
              return;
            }
            if (date) {
              setDateOfBirth(clampBirthDate(date));
            }
          }}
        />
      ) : null}

      <Modal
        visible={Platform.OS === 'ios' && showDobPicker}
        transparent
        animationType="slide"
        onRequestClose={() => setShowDobPicker(false)}>
        <View style={styles.dobModalRoot}>
          <Pressable
            style={styles.dobModalBackdropFill}
            onPress={() => setShowDobPicker(false)}
          />
          <View style={styles.dobModalSheet}>
            <View style={styles.dobModalToolbar}>
              <TouchableOpacity
                onPress={() => setShowDobPicker(false)}
                hitSlop={12}
                accessibilityRole="button"
                accessibilityLabel="Cancel">
                <Text style={styles.dobModalBtn}>Cancel</Text>
              </TouchableOpacity>
              <TouchableOpacity
                onPress={confirmIosDob}
                hitSlop={12}
                accessibilityRole="button"
                accessibilityLabel="Done">
                <Text style={[styles.dobModalBtn, styles.dobModalBtnPrimary]}>
                  Done
                </Text>
              </TouchableOpacity>
            </View>
            <DateTimePicker
              value={iosDobDraft}
              mode="date"
              display="spinner"
              themeVariant="dark"
              textColor="#ffffff"
              minimumDate={minBirthDate()}
              maximumDate={maxBirthDate()}
              onChange={(_, d) => {
                if (d) {
                  setIosDobDraft(clampBirthDate(d));
                }
              }}
            />
          </View>
        </View>
      </Modal>

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
            {POSITIONS.map((p) => (
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
    overflow: 'hidden',
  },
  photoDashedFilled: {
    borderStyle: 'solid',
    borderColor: '#6b7280',
  },
  photoAvatar: {
    ...StyleSheet.absoluteFillObject,
    width: '100%',
    height: '100%',
  },
  photoEditBadge: {
    position: 'absolute',
    bottom: 10,
    right: 10,
    width: 26,
    height: 26,
    borderRadius: 13,
    backgroundColor: LIME,
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1,
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
  phaseSection: {
    marginBottom: 8,
  },
  phaseTitle: {
    fontSize: 18,
    fontWeight: '800',
    color: WHITE,
    letterSpacing: 0.5,
    marginBottom: 8,
  },
  phaseHint: {
    fontSize: 13,
    color: MUTED,
    lineHeight: 19,
    marginBottom: 18,
  },
  traitRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: 12,
    paddingHorizontal: 12,
    marginBottom: 8,
    backgroundColor: CARD,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: BORDER,
  },
  traitRowSelected: {
    borderColor: LIME,
    backgroundColor: '#0f172a',
  },
  traitRowLeading: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    minWidth: 0,
  },
  traitRowLabel: {
    flex: 1,
    fontSize: 12,
    fontWeight: '800',
    color: LIME,
    letterSpacing: 0.35,
  },
  coreCard: {
    backgroundColor: '#0f172a',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#1e293b',
    padding: 12,
  },
  coreCardTitle: {
    fontSize: 14,
    fontWeight: '800',
    color: '#f8fafc',
    textTransform: 'uppercase',
    fontStyle: 'italic',
    letterSpacing: 0.9,
    marginBottom: 14,
  },
  attrRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  attrLabel: {
    fontSize: 11,
    color: '#94a3b8',
    fontWeight: '700',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  attrValueWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  attrBtn: {
    width: 24,
    height: 24,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#111827',
  },
  attrValueText: {
    minWidth: 28,
    textAlign: 'center',
    fontSize: 12,
    fontWeight: '800',
    color: '#e2e8f0',
  },
  attrBarTrack: {
    height: 6,
    borderRadius: 999,
    backgroundColor: '#1e293b',
    overflow: 'hidden',
    marginBottom: 12,
  },
  attrBarGapAfter: {
    marginBottom: 14,
  },
  attrBarFill: {
    height: '100%',
    borderRadius: 999,
    backgroundColor: '#d9f99d',
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
  dobHintRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: 8,
    marginTop: 6,
    marginLeft: 4,
  },
  dobHintText: {
    flex: 1,
    fontSize: 11,
    color: '#6b7280',
    lineHeight: 16,
  },
  dobClear: {
    fontSize: 11,
    fontWeight: '700',
    color: LIME,
    letterSpacing: 0.5,
    paddingTop: 1,
  },
  dobModalRoot: {
    flex: 1,
    justifyContent: 'flex-end',
  },
  dobModalBackdropFill: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0,0,0,0.55)',
  },
  dobModalSheet: {
    backgroundColor: '#0f172a',
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    paddingBottom: 28,
    borderTopWidth: 1,
    borderColor: BORDER,
  },
  dobModalToolbar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: BORDER,
  },
  dobModalBtn: {
    fontSize: 16,
    color: MUTED,
    fontWeight: '600',
  },
  dobModalBtnPrimary: {
    color: LIME,
    fontWeight: '800',
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
    color: WHITE,
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
