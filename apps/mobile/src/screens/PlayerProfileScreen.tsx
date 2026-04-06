import React, {useState, useEffect, useCallback} from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  ActivityIndicator,
  Image,
  Alert,
  TextInput,
  Switch,
  Dimensions,
} from 'react-native';
import {useNavigation, useFocusEffect} from '@react-navigation/native';
import type {NavigationProp} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {LinearGradient} from 'expo-linear-gradient';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {countryCodeToFlag, COUNTRIES} from '../data/countries';
import {userFacingApiMessage} from '../services/api';
import {
  getPlayerProfile,
  createPlayerProfile,
  updatePlayerProfile,
  deletePlayerProfile,
  setPlayerProfileTraits,
} from '../services/playerProfile';
import type {
  PlayerProfile as PlayerProfileType,
  PlayerProfileDraft,
  PlayerProfileOrDraft,
} from '../types/playerProfile';
import {CountryPicker} from '../components/CountryPicker';
import {PlayerTraitsModal} from '../components/PlayerTraitsModal';
import {PlayerTraitStripItem} from '../components/player_traits';
import {
  DEFAULT_CORE_RATING,
  defaultCoreAttrs,
  defaultDribblingDefending,
} from '../utils/playerCoreAttrs';
import {buildCompareSnapshotFromProfile} from '../utils/comparePlayerSnapshot';
import {displayLevel} from '../utils/playerRating';
import {useHoldCoreStep} from '../hooks/useHoldCoreStep';

type TabId = 'profile' | 'stats' | 'career';
type WorkRate = 'LOW' | 'MEDIUM' | 'HIGH';

const {width: SCREEN_WIDTH, height: SCREEN_HEIGHT} = Dimensions.get('window');
const HERO_HEIGHT = Math.max(340, SCREEN_HEIGHT * 0.42) * 0.75;
/** Hero card horizontal inset (matches `heroSection` marginHorizontal). */
const HERO_CARD_MARGIN = 14;

function roleArchetype(pos: PlayerProfileType['position'] | null): string {
  switch (pos) {
    case 'GK':
      return 'ELITE KEEPER';
    case 'DEF':
      return 'WALL DEFENDER';
    case 'MID':
      return 'PLAYMAKER';
    case 'FWD':
      return 'LEGENDARY STRIKER';
    default:
      return 'PROSPECT';
  }
}

const WORK_RATE_OPTIONS: WorkRate[] = ['LOW', 'MEDIUM', 'HIGH'];

export default function PlayerProfileScreen() {
  const navigation = useNavigation<NavigationProp>();
  const {token, user} = useAuth();
  const [profile, setProfile] = useState<PlayerProfileOrDraft | null>(null);
  const [isDraftProfile, setIsDraftProfile] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<TabId>('profile');
  const [showCountryPicker, setShowCountryPicker] = useState(false);
  const [showTraitsModal, setShowTraitsModal] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Edit form state (T014)
  const [editAge, setEditAge] = useState('');
  const [editCountryCode, setEditCountryCode] = useState<string | null>(null);
  const [editCountryName, setEditCountryName] = useState('');
  const [editClub, setEditClub] = useState('');
  const [editIsFreeAgent, setEditIsFreeAgent] = useState(false);
  const [editPosition, setEditPosition] = useState<
    PlayerProfileType['position'] | null
  >(null);
  const [editSpeed, setEditSpeed] = useState(DEFAULT_CORE_RATING);
  const [editShooting, setEditShooting] = useState(DEFAULT_CORE_RATING);
  const [editPassing, setEditPassing] = useState(DEFAULT_CORE_RATING);
  const [editPhysical, setEditPhysical] = useState(DEFAULT_CORE_RATING);
  const [editStamina, setEditStamina] = useState(DEFAULT_CORE_RATING);
  const [editAttackingWorkRate, setEditAttackingWorkRate] =
    useState<WorkRate>('MEDIUM');
  const [editDefensiveWorkRate, setEditDefensiveWorkRate] =
    useState<WorkRate>('LOW');
  const [editMode, setEditMode] = useState(false);
  const [showPositionPicker, setShowPositionPicker] = useState(false);

  const holdSpeedMinus = useHoldCoreStep(-1, setEditSpeed, saving);
  const holdSpeedPlus = useHoldCoreStep(1, setEditSpeed, saving);
  const holdShootingMinus = useHoldCoreStep(-1, setEditShooting, saving);
  const holdShootingPlus = useHoldCoreStep(1, setEditShooting, saving);
  const holdPassingMinus = useHoldCoreStep(-1, setEditPassing, saving);
  const holdPassingPlus = useHoldCoreStep(1, setEditPassing, saving);
  const holdPhysicalMinus = useHoldCoreStep(-1, setEditPhysical, saving);
  const holdPhysicalPlus = useHoldCoreStep(1, setEditPhysical, saving);
  const holdStaminaMinus = useHoldCoreStep(-1, setEditStamina, saving);
  const holdStaminaPlus = useHoldCoreStep(1, setEditStamina, saving);

  const POSITIONS: {value: PlayerProfileType['position']; label: string}[] = [
    {value: 'GK', label: 'Goalkeeper'},
    {value: 'DEF', label: 'Defender'},
    {value: 'MID', label: 'Midfielder'},
    {value: 'FWD', label: 'Forward'},
  ];

  const cycleWorkRate = (
    value: WorkRate,
    setter: (next: WorkRate) => void,
  ) => {
    const idx = WORK_RATE_OPTIONS.indexOf(value);
    const next = WORK_RATE_OPTIONS[(idx + 1) % WORK_RATE_OPTIONS.length];
    setter(next);
  };

  const loadProfile = useCallback(async () => {
    if (!token) {
      setLoading(false);
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const p = await getPlayerProfile(token);
      if (p) {
        setProfile(p);
        setIsDraftProfile(false);
        setEditAge(p.age != null ? String(p.age) : '');
        setEditCountryCode(p.country || null);
        const countryName = p.country
          ? (COUNTRIES.find(c => c.code === p.country)?.name ?? p.country)
          : '';
        setEditCountryName(countryName);
        setEditClub(p.club ?? '');
        setEditIsFreeAgent(p.is_free_agent ?? false);
        setEditPosition(p.position ?? null);
        setEditSpeed(p.speed);
        setEditShooting(p.shooting);
        setEditPassing(p.passing);
        setEditPhysical(p.physical);
        setEditStamina(p.stamina);
        setEditAttackingWorkRate('MEDIUM');
        setEditDefensiveWorkRate('LOW');
      } else {
        const draft: PlayerProfileDraft = {
          id: 0,
          age: null,
          country: '',
          club: null,
          is_free_agent: false,
          position: null,
          photo_url: null,
          speed: DEFAULT_CORE_RATING,
          shooting: DEFAULT_CORE_RATING,
          passing: DEFAULT_CORE_RATING,
          dribbling: DEFAULT_CORE_RATING,
          defending: DEFAULT_CORE_RATING,
          physical: DEFAULT_CORE_RATING,
          stamina: DEFAULT_CORE_RATING,
          traits: [],
          career_teams: [],
        };
        setProfile(draft);
        setIsDraftProfile(true);
        setEditAge('');
        setEditCountryCode(null);
        setEditCountryName('');
        setEditClub('');
        setEditIsFreeAgent(false);
        setEditPosition(null);
        const defaults = defaultCoreAttrs();
        setEditSpeed(defaults.speed);
        setEditShooting(defaults.shooting);
        setEditPassing(defaults.passing);
        setEditPhysical(defaults.physical);
        setEditStamina(defaults.stamina);
        setEditAttackingWorkRate('MEDIUM');
        setEditDefensiveWorkRate('LOW');
        setEditMode(false);
      }
    } catch {
      setError('Failed to load profile.');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useFocusEffect(
    useCallback(() => {
      loadProfile();
    }, [loadProfile]),
  );

  useEffect(() => {
    if (!token) {
      setLoading(false);
      setProfile(null);
    }
  }, [token]);

  const handleSaveProfile = async () => {
    if (!token) return;
    if (!editCountryCode || !editPosition) {
      setSaveError('Country and position are required.');
      return;
    }
    const ageNum = editAge.trim() ? parseInt(editAge.trim(), 10) : null;
    if (ageNum !== null && (ageNum < 13 || ageNum > 60)) {
      setSaveError('Age must be between 13 and 60.');
      return;
    }
    setSaveError(null);
    setSaving(true);
    try {
      const dd = defaultDribblingDefending();
      const payload = {
        country: editCountryCode,
        position: editPosition,
        age: ageNum,
        club: editIsFreeAgent ? null : editClub.trim() || null,
        is_free_agent: editIsFreeAgent,
        speed: editSpeed,
        shooting: editShooting,
        passing: editPassing,
        physical: editPhysical,
        stamina: editStamina,
        dribbling: dd.dribbling,
        defending: dd.defending,
      };
      const updated = isDraftProfile
        ? await createPlayerProfile(token, payload)
        : await updatePlayerProfile(token, payload);
      setProfile(updated);
      setIsDraftProfile(false);
      setEditMode(false);
    } catch (err) {
      setSaveError(userFacingApiMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteProfile = () => {
    Alert.alert(
      'Delete Player Profile?',
      'This will remove your player profile. You can create a new one later.',
      [
        {text: 'Cancel', style: 'cancel'},
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            if (!token) return;
            try {
              await deletePlayerProfile(token);
              const draft: PlayerProfileDraft = {
                id: 0,
                age: null,
                country: '',
                club: null,
                is_free_agent: false,
                position: null,
                photo_url: null,
                speed: DEFAULT_CORE_RATING,
                shooting: DEFAULT_CORE_RATING,
                passing: DEFAULT_CORE_RATING,
                dribbling: DEFAULT_CORE_RATING,
                defending: DEFAULT_CORE_RATING,
                physical: DEFAULT_CORE_RATING,
                stamina: DEFAULT_CORE_RATING,
                traits: [],
                career_teams: [],
              };
              setProfile(draft);
              setIsDraftProfile(true);
              setEditAge('');
              setEditCountryCode(null);
              setEditCountryName('');
              setEditClub('');
              setEditIsFreeAgent(false);
              setEditPosition(null);
              const defaults = defaultCoreAttrs();
              setEditSpeed(defaults.speed);
              setEditShooting(defaults.shooting);
              setEditPassing(defaults.passing);
              setEditPhysical(defaults.physical);
              setEditStamina(defaults.stamina);
              setEditAttackingWorkRate('MEDIUM');
              setEditDefensiveWorkRate('LOW');
              setEditMode(true);
            } catch (err) {
              setError(userFacingApiMessage(err));
            }
          },
        },
      ],
    );
  };

  const handleSaveTraits = async (traits: string[]) => {
    if (!token) return;
    setSaveError(null);
    if (isDraftProfile) {
      Alert.alert(
        'Complete Profile First',
        'Add age, country, and position before selecting traits.',
      );
      setShowTraitsModal(false);
      return;
    }
    try {
      const updated = await setPlayerProfileTraits(token, traits);
      if (profile) {
        setProfile({...profile, traits: updated});
      }
      setShowTraitsModal(false);
    } catch (err) {
      const msg = userFacingApiMessage(err);
      setSaveError(msg);
      Alert.alert('Save Failed', msg);
    }
  };

  if (loading) {
    return (
      <SafeAreaView style={styles.container} edges={['top']}>
        <View style={styles.loadingWrap}>
          <ActivityIndicator size="large" color="#22c55e" />
          <Text style={styles.loadingText}>Loading profile…</Text>
        </View>
      </SafeAreaView>
    );
  }

  if (error && !profile) {
    return (
      <SafeAreaView style={styles.container} edges={['top']}>
        <View style={styles.header}>
          <TouchableOpacity
            onPress={() => navigation.goBack()}
            style={styles.backBtn}>
            <Ionicons name="arrow-back" size={24} color="#e5e7eb" />
          </TouchableOpacity>
          <View style={styles.headerSpacer} />
        </View>
        <View style={styles.loadingWrap}>
          <Text style={styles.errorText}>{error}</Text>
          <TouchableOpacity style={styles.retryBtn} onPress={loadProfile}>
            <Text style={styles.retryBtnText}>Retry</Text>
          </TouchableOpacity>
        </View>
      </SafeAreaView>
    );
  }

  if (!profile) {
    return null;
  }

  const flag =
    editCountryCode || profile.country
      ? countryCodeToFlag(editCountryCode || profile.country)
      : '';
  const countryName =
    COUNTRIES.find(c => c.code === profile.country)?.name ?? profile.country;

  const posAbbrev =
    profile.position == null
      ? ''
      : profile.position === 'GK'
        ? 'GK'
        : profile.position === 'DEF'
          ? 'CB'
          : profile.position === 'MID'
            ? 'CM'
            : 'ST';

  const clubOrStatus = profile.is_free_agent
    ? 'Free Agent'
    : profile.club?.trim()
      ? profile.club.trim()
      : '—';

  const ageComplete = profile.age != null;
  const countryComplete = !!profile.country;
  const positionComplete = !isDraftProfile && !!profile.position;
  const traitsComplete = !isDraftProfile && (profile.traits?.length ?? 0) > 0;
  const completionCount =
    Number(ageComplete) +
    Number(countryComplete) +
    Number(positionComplete) +
    Number(traitsComplete);
  const completionPercent = Math.round((completionCount / 4) * 100);

  const canGoBack = navigation.canGoBack();
  const avatarUri = profile.photo_url || user?.avatar_url || null;
  const coreAttrs = isDraftProfile
    ? defaultCoreAttrs()
    : {
        speed: profile.speed,
        shooting: profile.shooting,
        passing: profile.passing,
        physical: profile.physical,
        stamina: profile.stamina,
      };
  const displayNameRaw =
    user?.display_name?.trim() ||
    [user?.firstname, user?.lastname].filter(Boolean).join(' ').trim() ||
    'Player';
  const barTitle = displayNameRaw.replace(/\s+/g, '_').toUpperCase();
  const heroLevel = displayLevel(
    profile.traits?.length ?? 0,
    completionPercent,
  );
  const heroSubtitle = `${roleArchetype(isDraftProfile ? null : profile.position)} • LEVEL ${heroLevel}`;

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.topBar}>
        <View style={styles.topBarLeft}>
          {canGoBack ? (
            <TouchableOpacity
              onPress={() => navigation.goBack()}
              style={styles.topBarBack}
              accessibilityLabel="Go back">
              <Ionicons name="chevron-back" size={22} color="#94a3b8" />
            </TouchableOpacity>
          ) : null}
        </View>
        <View style={styles.topBarRight}>
          {!isDraftProfile && profile.position ? (
            <TouchableOpacity
              style={styles.topBarCompare}
              onPress={() => {
                const snap = buildCompareSnapshotFromProfile(profile, user, {
                  isDraftProfile,
                  traitsLen: profile.traits?.length ?? 0,
                  completionPercent,
                });
                if (!snap) {
                  Alert.alert(
                    'Complete your profile',
                    'Add a position and save your profile before comparing.',
                  );
                  return;
                }
                navigation.navigate('PlayerCompare', {left: snap});
              }}
              accessibilityLabel="Compare players">
              <Ionicons
                name="git-compare-outline"
                size={18}
                color="#c7f349"
              />
              <Text style={styles.topBarCompareText}>Compare</Text>
            </TouchableOpacity>
          ) : null}
        </View>
      </View>

      {completionPercent < 100 && !editMode ? (
        <View style={styles.progressMini}>
          <View style={styles.progressTrackMini}>
            <View
              style={[
                styles.progressFillMini,
                {width: `${completionPercent}%`},
              ]}
            />
          </View>
          <Text style={styles.progressMiniLabel}>{completionPercent}%</Text>
        </View>
      ) : null}

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>
        <View style={[styles.heroSection, {height: HERO_HEIGHT}]}>
          <View style={styles.player_image_cutout}>
            <Image
              source={require('../../assets/hero_stadium.jpg')}
              style={styles.playerCutoutImage}
              resizeMode="cover"
            />
          </View>
          <LinearGradient
            colors={[
              'rgba(5,8,20,0.35)',
              'rgba(5,8,20,0.88)',
              'rgba(5,8,20,1)',
            ]}
            locations={[0, 0.55, 1]}
            style={styles.heroNightOverlay}
            pointerEvents="none"
          />
          <LinearGradient
            colors={[
              'transparent',
              'rgba(132,204,22,0.08)',
              'rgba(74,222,128,0.12)',
            ]}
            locations={[0.4, 0.78, 1]}
            style={styles.heroGradientFade}
            pointerEvents="none"
          />
          <View style={styles.heroIdentityBlock}>
            <View style={styles.heroAvatarCard}>
              <View style={styles.heroAvatarGold}>
                {avatarUri ? (
                  <Image
                    source={{uri: avatarUri}}
                    style={styles.heroAvatarImg}
                  />
                ) : (
                  <View style={styles.heroAvatarPlaceholder}>
                    <Ionicons name="person" size={40} color="#64748b" />
                  </View>
                )}
              </View>
            </View>
            <Text style={styles.heroDisplayName} numberOfLines={2}>
              {displayNameRaw.toUpperCase()}
            </Text>
            {completionPercent < 100 && !editMode ? (
              <TouchableOpacity onPress={() => setEditMode(true)}>
                <Text style={styles.heroCompletionHint}>
                  Profile {completionPercent}% complete — tap settings to finish
                </Text>
              </TouchableOpacity>
            ) : null}
          </View>
        </View>

        <View style={styles.tabsSticky}>
          {(['profile', 'stats', 'career'] as const).map(tab => (
            <TouchableOpacity
              key={tab}
              style={[styles.tab, activeTab === tab && styles.tabActive]}
              onPress={() => setActiveTab(tab)}
              activeOpacity={0.7}>
              <Text
                style={[
                  styles.tabText,
                  activeTab === tab && styles.tabTextActive,
                ]}>
                {tab.toUpperCase()}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {activeTab === 'profile' && (
          <>
            {!editMode ? (
              <>
                <View style={styles.infoGrid}>
                  <View style={styles.infoTile}>
                    <Text style={styles.infoTileLabel}>Age</Text>
                    <Text style={styles.infoTileValue}>
                      {profile.age ?? '—'}
                    </Text>
                  </View>
                  <View style={styles.infoTile}>
                    <Text style={styles.infoTileLabel}>Country</Text>
                    <Text style={styles.infoTileValue} numberOfLines={2}>
                      {flag ? `${flag} ` : ''}
                      {countryName || '—'}
                    </Text>
                  </View>
                  <View style={styles.infoTile}>
                    <Text style={styles.infoTileLabel}>Team</Text>
                    <Text style={styles.infoTileValue} numberOfLines={2}>
                      {clubOrStatus}
                    </Text>
                  </View>
                  <View style={styles.infoTile}>
                    <Text style={styles.infoTileLabel}>Position</Text>
                    <Text
                      style={[styles.infoTileValue, styles.infoTileValueCyan]}>
                      {posAbbrev || '—'}
                    </Text>
                  </View>
                </View>

                <View style={styles.traitsSection}>
                  <View style={styles.traitsHeader}>
                    <View>
                      <Text style={styles.traitsTitle}>Player traits</Text>
                      <Text style={styles.traitsSubtitle}>
                        Mastered skills & abilities
                      </Text>
                    </View>
                    <TouchableOpacity
                      style={styles.viewAllTraits}
                      onPress={() => setShowTraitsModal(true)}>
                      <Text style={styles.viewAllTraitsText}>View all</Text>
                    </TouchableOpacity>
                  </View>
                  {(profile.traits ?? []).length === 0 ? (
                    <View style={styles.traitsEmptyCard}>
                      <Text style={styles.traitsEmpty}>No traits yet</Text>
                      <Text style={styles.traitsEmptyHint}>
                        Tap View all to choose up to 5
                      </Text>
                    </View>
                  ) : (
                    <View style={styles.traitsGrid}>
                      {(profile.traits ?? []).map((code: string) => (
                        <View key={code} style={styles.traitGridCell}>
                          <PlayerTraitStripItem code={code} />
                        </View>
                      ))}
                    </View>
                  )}
                </View>

                <View style={styles.coreCard}>
                  <View style={styles.coreCardHeader}>
                    <Text style={styles.coreCardTitle}>Core attributes</Text>
                    <View style={styles.coreDots}>
                      <View style={[styles.coreDot, styles.coreDotActive]} />
                      <View style={styles.coreDot} />
                    </View>
                  </View>
                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Speed</Text>
                    <Text style={styles.attrNum}>{coreAttrs.speed}</Text>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${coreAttrs.speed}%`},
                      ]}
                    />
                  </View>
                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Shooting</Text>
                    <Text style={styles.attrNum}>{coreAttrs.shooting}</Text>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${coreAttrs.shooting}%`},
                      ]}
                    />
                  </View>
                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Passing</Text>
                    <Text style={styles.attrNum}>{coreAttrs.passing}</Text>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${coreAttrs.passing}%`},
                      ]}
                    />
                  </View>
                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Physical</Text>
                    <Text style={styles.attrNum}>{coreAttrs.physical}</Text>
                  </View>
                  <View style={styles.attrBarTrack}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${coreAttrs.physical}%`},
                      ]}
                    />
                  </View>
                  <View style={styles.attrRow}>
                    <Text style={styles.attrLabel}>Stamina</Text>
                    <Text style={styles.attrNum}>{coreAttrs.stamina}</Text>
                  </View>
                  <View style={[styles.attrBarTrack, styles.attrBarTrackLast]}>
                    <View
                      style={[
                        styles.attrBarFill,
                        {width: `${coreAttrs.stamina}%`},
                      ]}
                    />
                  </View>
                </View>

                <View style={styles.careerSection}>
                  <Text style={styles.sectionTitle}>Career teams</Text>
                  {profile.career_teams && profile.career_teams.length > 0 ? (
                    profile.career_teams
                      .slice()
                      .sort((a, b) => b.start_year - a.start_year)
                      .map((ct, idx) => (
                        <View
                          key={ct.id}
                          style={[
                            styles.careerRow,
                            idx === profile.career_teams!.length - 1 &&
                              styles.careerRowLast,
                          ]}>
                          <View style={styles.careerLogo}>
                            <Ionicons
                              name="shield-outline"
                              size={24}
                              color="#6b7280"
                            />
                          </View>
                          <View style={styles.careerInfo}>
                            <Text style={styles.careerTeamName}>
                              {ct.team_name}
                            </Text>
                            <Text style={styles.careerTenure}>
                              {ct.start_year} – {ct.end_year ?? 'Present'}
                            </Text>
                          </View>
                          <Ionicons
                            name="chevron-forward"
                            size={20}
                            color="#9ca3af"
                          />
                        </View>
                      ))
                  ) : (
                    <Text style={styles.careerEmpty}>No career teams yet.</Text>
                  )}
                </View>

                <TouchableOpacity
                  style={styles.editLink}
                  onPress={() => setEditMode(true)}>
                  <Text style={styles.editLinkText}>Edit profile details</Text>
                </TouchableOpacity>

                {!isDraftProfile ? (
                  <TouchableOpacity
                    style={styles.deleteBtn}
                    onPress={handleDeleteProfile}>
                    <Text style={styles.deleteBtnText}>
                      Delete player profile
                    </Text>
                  </TouchableOpacity>
                ) : null}
              </>
            ) : null}

            {editMode ? (
              <View style={styles.editFormCard}>
                <View style={styles.editForm}>
                  <Text style={styles.label}>Age</Text>
                  <TextInput
                    style={styles.input}
                    value={editAge}
                    onChangeText={setEditAge}
                    placeholder="13–60"
                    placeholderTextColor="#64748b"
                    keyboardType="number-pad"
                    maxLength={2}
                    editable={!saving}
                  />
                </View>
                <TouchableOpacity
                  style={styles.editFormRow}
                  onPress={() => setShowCountryPicker(true)}>
                  <Text style={styles.label}>Country</Text>
                  <View style={styles.rowValue}>
                    {flag ? <Text style={styles.flag}>{flag}</Text> : null}
                    <Text style={styles.value}>
                      {editCountryName || 'Select'}
                    </Text>
                    <Ionicons
                      name="chevron-forward"
                      size={18}
                      color="#94a3b8"
                    />
                  </View>
                </TouchableOpacity>
                <View style={styles.editForm}>
                  <Text style={styles.label}>Club or Free Agent</Text>
                  <TextInput
                    style={[
                      styles.input,
                      editIsFreeAgent && styles.inputDisabled,
                    ]}
                    value={editClub}
                    onChangeText={setEditClub}
                    placeholder="Club name"
                    placeholderTextColor="#64748b"
                    editable={!editIsFreeAgent && !saving}
                  />
                  <View style={styles.toggleRow}>
                    <Text style={styles.toggleLabel}>Free Agent</Text>
                    <Switch
                      value={editIsFreeAgent}
                      onValueChange={setEditIsFreeAgent}
                      trackColor={{false: '#334155', true: '#84cc16'}}
                      thumbColor={editIsFreeAgent ? '#0f172a' : '#e2e8f0'}
                      disabled={saving}
                    />
                  </View>
                </View>
                <TouchableOpacity
                  style={styles.editFormRow}
                  onPress={() => setShowPositionPicker(true)}>
                  <Text style={styles.label}>Position</Text>
                  <View style={styles.rowValue}>
                    <Text style={styles.value}>
                      {editPosition
                        ? (POSITIONS.find(p => p.value === editPosition)
                            ?.label ?? editPosition)
                        : 'Select'}
                    </Text>
                    <Ionicons
                      name="chevron-forward"
                      size={18}
                      color="#94a3b8"
                    />
                  </View>
                </TouchableOpacity>
                <View style={styles.editCoreCard}>
                  <Text style={styles.editCoreTitle}>Core attributes</Text>

                  <View style={styles.editAttrRow}>
                    <Text style={styles.editAttrLabel}>Speed</Text>
                    <View style={styles.editAttrValueWrap}>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdSpeedMinus.onPressIn}
                        onPressOut={holdSpeedMinus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.editAttrValueText}>{editSpeed}</Text>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdSpeedPlus.onPressIn}
                        onPressOut={holdSpeedPlus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.editAttrBarTrack}>
                    <View
                      style={[styles.editAttrBarFill, {width: `${editSpeed}%`}]}
                    />
                  </View>

                  <View style={styles.editAttrRow}>
                    <Text style={styles.editAttrLabel}>Shooting</Text>
                    <View style={styles.editAttrValueWrap}>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdShootingMinus.onPressIn}
                        onPressOut={holdShootingMinus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.editAttrValueText}>
                        {editShooting}
                      </Text>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdShootingPlus.onPressIn}
                        onPressOut={holdShootingPlus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.editAttrBarTrack}>
                    <View
                      style={[
                        styles.editAttrBarFill,
                        {width: `${editShooting}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.editAttrRow}>
                    <Text style={styles.editAttrLabel}>Passing</Text>
                    <View style={styles.editAttrValueWrap}>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdPassingMinus.onPressIn}
                        onPressOut={holdPassingMinus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.editAttrValueText}>{editPassing}</Text>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdPassingPlus.onPressIn}
                        onPressOut={holdPassingPlus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={styles.editAttrBarTrack}>
                    <View
                      style={[
                        styles.editAttrBarFill,
                        {width: `${editPassing}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.editAttrRow}>
                    <Text style={styles.editAttrLabel}>Physical</Text>
                    <View style={styles.editAttrValueWrap}>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdPhysicalMinus.onPressIn}
                        onPressOut={holdPhysicalMinus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.editAttrValueText}>{editPhysical}</Text>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdPhysicalPlus.onPressIn}
                        onPressOut={holdPhysicalPlus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={[styles.editAttrBarTrack, styles.editAttrBarLast]}>
                    <View
                      style={[
                        styles.editAttrBarFill,
                        {width: `${editPhysical}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.editAttrRow}>
                    <Text style={styles.editAttrLabel}>Stamina</Text>
                    <View style={styles.editAttrValueWrap}>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdStaminaMinus.onPressIn}
                        onPressOut={holdStaminaMinus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="remove" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                      <Text style={styles.editAttrValueText}>{editStamina}</Text>
                      <TouchableOpacity
                        style={styles.editAttrBtn}
                        onPressIn={holdStaminaPlus.onPressIn}
                        onPressOut={holdStaminaPlus.onPressOut}
                        disabled={saving}
                        delayPressIn={0}>
                        <Ionicons name="add" size={14} color="#e2e8f0" />
                      </TouchableOpacity>
                    </View>
                  </View>
                  <View style={[styles.editAttrBarTrack, styles.editAttrBarLast]}>
                    <View
                      style={[
                        styles.editAttrBarFill,
                        {width: `${editStamina}%`},
                      ]}
                    />
                  </View>

                  <View style={styles.editWorkRateRow}>
                    <TouchableOpacity
                      style={styles.editWorkRateCard}
                      onPress={() =>
                        cycleWorkRate(
                          editAttackingWorkRate,
                          setEditAttackingWorkRate,
                        )
                      }
                      disabled={saving}>
                      <Text style={styles.editWorkRateLabel}>
                        Attacking Workrate
                      </Text>
                      <View style={styles.editWorkRateValueRow}>
                        <Text style={styles.editWorkRateValue}>
                          {editAttackingWorkRate}
                        </Text>
                        <Ionicons
                          name="chevron-down"
                          size={14}
                          color="#94a3b8"
                        />
                      </View>
                    </TouchableOpacity>

                    <TouchableOpacity
                      style={styles.editWorkRateCard}
                      onPress={() =>
                        cycleWorkRate(
                          editDefensiveWorkRate,
                          setEditDefensiveWorkRate,
                        )
                      }
                      disabled={saving}>
                      <Text style={styles.editWorkRateLabel}>
                        Defensive Workrate
                      </Text>
                      <View style={styles.editWorkRateValueRow}>
                        <Text style={styles.editWorkRateValue}>
                          {editDefensiveWorkRate}
                        </Text>
                        <Ionicons
                          name="chevron-down"
                          size={14}
                          color="#94a3b8"
                        />
                      </View>
                    </TouchableOpacity>
                  </View>
                </View>
              </View>
            ) : null}

            {editMode ? (
              <>
                {saveError ? (
                  <Text style={styles.saveError}>{saveError}</Text>
                ) : null}
                <TouchableOpacity
                  style={[
                    styles.ctaBtn,
                    styles.ctaBtnPrimary,
                    saving && styles.saveBtnDisabled,
                  ]}
                  onPress={handleSaveProfile}
                  disabled={saving}>
                  {saving ? (
                    <ActivityIndicator color="#fff" size="small" />
                  ) : (
                    <Text style={styles.ctaBtnText}>SAVE PROFILE</Text>
                  )}
                </TouchableOpacity>
                <TouchableOpacity
                  style={styles.cancelEditBtn}
                  onPress={() => setEditMode(false)}
                  disabled={saving}>
                  <Text style={styles.cancelEditText}>Cancel</Text>
                </TouchableOpacity>
              </>
            ) : null}
          </>
        )}

        {activeTab === 'stats' && (
          <View style={styles.tabPanel}>
            <View style={styles.placeholderInner}>
              <Ionicons name="stats-chart-outline" size={48} color="#6b7280" />
              <Text style={styles.placeholderText}>Stats coming soon</Text>
            </View>
          </View>
        )}

        {activeTab === 'career' && (
          <View style={styles.tabPanel}>
            <View style={styles.placeholderInner}>
              <Ionicons name="list-outline" size={48} color="#6b7280" />
              <Text style={styles.placeholderText}>
                Career timeline coming soon
              </Text>
            </View>
          </View>
        )}
      </ScrollView>

      <CountryPicker
        visible={showCountryPicker}
        onDismiss={() => setShowCountryPicker(false)}
        onSelect={(code, name) => {
          setEditCountryCode(code);
          setEditCountryName(name);
        }}
        selectedCode={editCountryCode}
      />

      <PlayerTraitsModal
        visible={showTraitsModal}
        onDismiss={() => setShowTraitsModal(false)}
        selectedTraits={profile.traits ?? []}
        onSave={handleSaveTraits}
      />

      {showPositionPicker ? (
        <View style={styles.positionOverlay}>
          <TouchableOpacity
            style={StyleSheet.absoluteFill}
            activeOpacity={1}
            onPress={() => setShowPositionPicker(false)}
          />
          <View style={styles.positionSheet}>
            <Text style={styles.positionSheetTitle}>Position</Text>
            {POSITIONS.map(p => (
              <TouchableOpacity
                key={p.value}
                style={styles.positionOption}
                onPress={() => {
                  setEditPosition(p.value);
                  const defaults = defaultCoreAttrs();
                  setEditSpeed(defaults.speed);
                  setEditShooting(defaults.shooting);
                  setEditPassing(defaults.passing);
                  setEditPhysical(defaults.physical);
                  setEditStamina(defaults.stamina);
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
    backgroundColor: '#0a0e17',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#030712',
    borderBottomWidth: 1,
    borderBottomColor: '#111827',
  },
  backBtn: {padding: 4},
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 17,
    fontWeight: '700',
    color: '#e5e7eb',
  },
  title: {fontSize: 18, fontWeight: '700', color: '#111827'},
  headerIconWrap: {
    width: 32,
    height: 32,
    alignItems: 'center',
    justifyContent: 'center',
    position: 'relative',
  },
  headerBadge: {
    position: 'absolute',
    top: 0,
    right: 0,
    width: 10,
    height: 10,
    borderRadius: 5,
    backgroundColor: '#ef4444',
    borderWidth: 1.5,
    borderColor: '#030712',
  },
  headerSpacer: {width: 32},
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 12,
    paddingVertical: 10,
    backgroundColor: '#0a0e17',
    borderBottomWidth: 1,
    borderBottomColor: '#151b2e',
  },
  topBarLeft: {
    flexDirection: 'row',
    alignItems: 'center',
    minWidth: 88,
    flexShrink: 0,
    gap: 4,
  },
  topBarRight: {
    flexShrink: 0,
    alignItems: 'flex-end',
    justifyContent: 'center',
  },
  topBarCompare: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
    paddingVertical: 6,
    paddingHorizontal: 10,
    borderRadius: 10,
    backgroundColor: 'rgba(199,243,73,0.12)',
    borderWidth: 1,
    borderColor: 'rgba(199,243,73,0.35)',
  },
  topBarCompareText: {
    fontSize: 12,
    fontWeight: '800',
    color: '#c7f349',
    letterSpacing: 0.4,
  },
  topBarBack: {padding: 4, marginRight: 0},
  topBarAvatar: {
    width: 34,
    height: 34,
    borderRadius: 17,
    backgroundColor: '#1a2235',
    overflow: 'hidden',
    alignItems: 'center',
    justifyContent: 'center',
    borderWidth: 1,
    borderColor: '#2d3a52',
  },
  topBarAvatarImg: {width: '100%', height: '100%'},
  topBarTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 12,
    fontWeight: '900',
    fontStyle: 'italic',
    color: '#a3e635',
    letterSpacing: 0.6,
    paddingHorizontal: 4,
  },
  topBarGear: {
    width: 44,
    flexShrink: 0,
    alignItems: 'flex-end',
    padding: 4,
  },
  progressMini: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    paddingHorizontal: 14,
    paddingVertical: 6,
    backgroundColor: '#0a0e17',
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1e293b',
  },
  progressTrackMini: {
    flex: 1,
    height: 4,
    borderRadius: 999,
    backgroundColor: '#1f2937',
    overflow: 'hidden',
  },
  progressFillMini: {
    height: '100%',
    backgroundColor: '#84cc16',
    borderRadius: 999,
  },
  progressMiniLabel: {
    fontSize: 11,
    fontWeight: '800',
    color: '#84cc16',
    minWidth: 34,
    textAlign: 'right',
  },
  tabsSticky: {
    flexDirection: 'row',
    flexWrap: 'nowrap',
    alignItems: 'stretch',
    width: '100%',
    alignSelf: 'stretch',
    backgroundColor: '#0a0e17',
    borderBottomWidth: 1,
    borderBottomColor: '#1e293b',
    zIndex: 4,
    elevation: 4,
  },
  tab: {
    flex: 1,
    flexBasis: 0,
    minWidth: 0,
    paddingVertical: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  tabActive: {
    borderBottomWidth: 2,
    borderBottomColor: '#84cc16',
  },
  tabText: {
    fontSize: 13,
    color: '#64748b',
    fontWeight: '700',
    letterSpacing: 0.6,
  },
  tabTextActive: {fontSize: 13, fontWeight: '800', color: '#84cc16'},
  tabPanel: {
    paddingHorizontal: 14,
    paddingTop: 20,
    paddingBottom: 32,
    minHeight: 220,
  },
  placeholderInner: {
    alignItems: 'center',
    justifyContent: 'center',
    paddingVertical: 40,
  },
  scroll: {flex: 1},
  scrollContent: {paddingBottom: 48},
  loadingWrap: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  loadingText: {marginTop: 12, color: '#6b7280'},
  errorText: {color: '#dc2626', textAlign: 'center'},
  retryBtn: {marginTop: 16, paddingVertical: 10, paddingHorizontal: 20},
  retryBtnText: {color: '#22c55e', fontWeight: '600'},
  heroSection: {
    width: SCREEN_WIDTH - HERO_CARD_MARGIN * 2,
    alignSelf: 'center',
    marginTop: 10,
    marginBottom: 6,
    borderRadius: 18,
    overflow: 'hidden',
    position: 'relative',
    borderWidth: 1,
    borderColor: 'rgba(30, 41, 59, 0.9)',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 10},
    shadowOpacity: 0.35,
    shadowRadius: 20,
    elevation: 8,
  },
  heroIdentityBlock: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    paddingBottom: 22,
    paddingHorizontal: 16,
    alignItems: 'center',
  },
  heroAvatarCard: {
    padding: 6,
    borderRadius: 16,
    backgroundColor: 'rgba(15, 23, 42, 0.55)',
    marginBottom: 12,
  },
  heroAvatarGold: {
    width: 92,
    height: 92,
    borderRadius: 46,
    borderWidth: 3,
    borderColor: '#f59e0b',
    overflow: 'hidden',
    backgroundColor: '#1e293b',
    alignItems: 'center',
    justifyContent: 'center',
  },
  heroAvatarImg: {width: '100%', height: '100%'},
  heroAvatarPlaceholder: {
    flex: 1,
    width: '100%',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#1e293b',
  },
  heroDisplayName: {
    fontSize: 22,
    fontWeight: '900',
    color: '#f8fafc',
    letterSpacing: 0.6,
    textAlign: 'center',
  },
  heroSubtitle: {
    marginTop: 6,
    fontSize: 12,
    fontWeight: '800',
    color: '#e2e8f0',
    letterSpacing: 1.2,
    textAlign: 'center',
  },
  heroCompletionHint: {
    marginTop: 10,
    fontSize: 11,
    fontWeight: '600',
    color: '#22d3ee',
    textAlign: 'center',
  },
  player_image_cutout: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
    width: '100%',
    overflow: 'hidden',
  },
  /** Full screen width, shifted so the crop is horizontally centered in the card. */
  playerCutoutImage: {
    position: 'absolute',
    top: 0,
    bottom: 0,
    left: -HERO_CARD_MARGIN,
    width: SCREEN_WIDTH,
    height: '100%',
  },
  heroGradientFade: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    height: 190,
  },
  heroNightOverlay: {
    ...StyleSheet.absoluteFillObject,
  },
  infoGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    rowGap: 10,
    marginHorizontal: 14,
    marginTop: 16,
    marginBottom: 8,
  },
  infoTile: {
    width: '48%',
    backgroundColor: '#141b2d',
    borderRadius: 12,
    paddingVertical: 14,
    paddingHorizontal: 10,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  infoTileLabel: {
    fontSize: 10,
    fontWeight: '700',
    color: '#64748b',
    letterSpacing: 1,
    textTransform: 'uppercase',
    marginBottom: 6,
    textAlign: 'center',
  },
  infoTileValue: {
    fontSize: 15,
    fontWeight: '800',
    color: '#f1f5f9',
    textAlign: 'center',
  },
  infoTileValueCyan: {
    color: '#22d3ee',
    fontSize: 17,
  },
  coreCard: {
    marginHorizontal: 14,
    marginBottom: 20,
    backgroundColor: '#141b2d',
    borderRadius: 16,
    padding: 16,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  coreCardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 16,
  },
  coreCardTitle: {
    fontSize: 14,
    fontWeight: '800',
    fontStyle: 'italic',
    color: '#f8fafc',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  coreDots: {flexDirection: 'row', gap: 6},
  coreDot: {
    width: 6,
    height: 6,
    borderRadius: 3,
    backgroundColor: '#334155',
  },
  coreDotActive: {backgroundColor: '#84cc16'},
  attrRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'baseline',
    marginBottom: 4,
  },
  attrLabel: {
    fontSize: 12,
    fontWeight: '700',
    color: '#94a3b8',
    letterSpacing: 0.5,
    textTransform: 'uppercase',
  },
  attrNum: {
    fontSize: 13,
    fontWeight: '800',
    color: '#f8fafc',
  },
  attrBarTrack: {
    height: 8,
    borderRadius: 999,
    backgroundColor: '#1e293b',
    overflow: 'hidden',
    marginBottom: 14,
  },
  attrBarTrackLast: {marginBottom: 18},
  attrBarFill: {
    height: '100%',
    borderRadius: 999,
    backgroundColor: '#84cc16',
  },
  upgradeBtn: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
  },
  upgradeBtnText: {
    fontSize: 14,
    fontWeight: '900',
    color: '#0f172a',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  traitsTitle: {
    fontSize: 14,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  traitsSubtitle: {
    marginTop: 4,
    fontSize: 10,
    fontWeight: '700',
    color: '#64748b',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  viewAllTraits: {paddingVertical: 4, paddingLeft: 8},
  viewAllTraitsText: {
    fontSize: 12,
    fontWeight: '800',
    color: '#22d3ee',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
  },
  editForm: {marginBottom: 16},
  editFormRow: {
    marginBottom: 16,
    backgroundColor: '#020617',
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#1f2937',
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  editCoreCard: {
    marginTop: 4,
    backgroundColor: '#0f172a',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#1e293b',
    padding: 12,
  },
  editCoreTitle: {
    fontSize: 14,
    fontWeight: '800',
    color: '#f8fafc',
    textTransform: 'uppercase',
    fontStyle: 'italic',
    letterSpacing: 0.9,
    marginBottom: 14,
  },
  editAttrRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  editAttrLabel: {
    fontSize: 11,
    color: '#94a3b8',
    fontWeight: '700',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  editAttrValueWrap: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  editAttrBtn: {
    width: 24,
    height: 24,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#111827',
  },
  editAttrValueText: {
    minWidth: 28,
    textAlign: 'center',
    fontSize: 12,
    fontWeight: '800',
    color: '#e2e8f0',
  },
  editAttrBarTrack: {
    height: 6,
    borderRadius: 999,
    backgroundColor: '#1e293b',
    overflow: 'hidden',
    marginBottom: 12,
  },
  editAttrBarLast: {
    marginBottom: 14,
  },
  editAttrBarFill: {
    height: '100%',
    borderRadius: 999,
    backgroundColor: '#d9f99d',
  },
  editWorkRateRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    gap: 10,
  },
  editWorkRateCard: {
    flex: 1,
    backgroundColor: '#020617',
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#1f2937',
    paddingHorizontal: 10,
    paddingVertical: 9,
  },
  editWorkRateLabel: {
    fontSize: 10,
    color: '#64748b',
    letterSpacing: 0.7,
    textTransform: 'uppercase',
    fontWeight: '700',
  },
  editWorkRateValueRow: {
    marginTop: 6,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  editWorkRateValue: {
    fontSize: 12,
    color: '#e2e8f0',
    fontWeight: '800',
    letterSpacing: 0.3,
  },
  field: {marginBottom: 16},
  label: {
    fontSize: 11,
    color: '#94a3b8',
    marginBottom: 6,
    fontWeight: '700',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  value: {fontSize: 18, color: '#f8fafc', fontWeight: '600'},
  hint: {fontSize: 12, color: '#9ca3af', marginTop: 2},
  row: {marginBottom: 16},
  rowValue: {flexDirection: 'row', alignItems: 'center'},
  flag: {fontSize: 20, marginRight: 8},
  traitsSection: {
    marginHorizontal: 14,
    marginBottom: 20,
    backgroundColor: '#141b2d',
    borderRadius: 14,
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  traitsHeader: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    marginBottom: 14,
  },
  sectionTitle: {
    fontSize: 15,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.8,
    textTransform: 'uppercase',
  },
  addTraitsBtn: {flexDirection: 'row', alignItems: 'center', gap: 6},
  addTraitsText: {fontSize: 12, color: '#22d3ee', fontWeight: '700'},
  traitsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    width: '100%',
    alignSelf: 'stretch',
    marginTop: 8,
    rowGap: 12,
    justifyContent: 'space-between',
  },
  traitGridCell: {
    flexBasis: '32%',
    flexGrow: 0,
    flexShrink: 0,
    maxWidth: '32%',
    alignItems: 'center',
    overflow: 'hidden',
  },
  traitsEmptyCard: {
    width: '100%',
    paddingVertical: 24,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 12,
    backgroundColor: '#0f172a',
    borderWidth: 1,
    borderColor: '#334155',
    borderStyle: 'dashed',
  },
  traitsEmpty: {fontSize: 14, color: '#cbd5e1'},
  traitsEmptyHint: {fontSize: 12, color: '#94a3b8', marginTop: 4},
  careerSection: {
    marginHorizontal: 14,
    marginBottom: 24,
    backgroundColor: '#141b2d',
    borderRadius: 14,
    padding: 16,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  careerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1f2937',
  },
  careerRowLast: {
    borderBottomWidth: 0,
  },
  editFormCard: {
    backgroundColor: '#141b2d',
    borderRadius: 14,
    padding: 16,
    marginHorizontal: 14,
    marginBottom: 20,
    marginTop: 8,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  careerLogo: {
    width: 44,
    height: 44,
    borderRadius: 10,
    backgroundColor: '#111827',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 14,
  },
  careerInfo: {flex: 1},
  careerTeamName: {fontSize: 14, fontWeight: '700', color: '#e2e8f0'},
  careerTenure: {fontSize: 12, color: '#93c5fd', marginTop: 2},
  careerEmpty: {fontSize: 14, color: '#94a3b8', paddingVertical: 12},
  saveError: {color: '#dc2626', fontSize: 14, marginBottom: 8},
  ctaBtn: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  ctaBtnPrimary: {
    backgroundColor: '#84cc16',
    shadowColor: '#65a30d',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.4,
    shadowRadius: 10,
    elevation: 6,
  },
  ctaBtnText: {
    color: '#0f172a',
    fontSize: 14,
    fontWeight: '900',
    letterSpacing: 1.1,
  },
  editLink: {alignItems: 'center', marginTop: 14, marginHorizontal: 14},
  editLinkText: {fontSize: 14, color: '#22d3ee', fontWeight: '700'},
  saveBtn: {
    backgroundColor: '#16a34a',
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 16,
  },
  saveBtnDisabled: {opacity: 0.7},
  saveBtnText: {color: '#fff', fontWeight: '700'},
  cancelEditBtn: {alignItems: 'center', marginTop: 12},
  cancelEditText: {fontSize: 15, color: '#94a3b8'},
  deleteBtn: {alignItems: 'center', marginTop: 24, marginHorizontal: 14},
  deleteBtnText: {fontSize: 13, color: '#f87171'},
  placeholder: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  placeholderText: {marginTop: 12, color: '#9ca3af'},
  input: {
    borderWidth: 1,
    borderColor: '#1f2937',
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 18,
    color: '#f8fafc',
    backgroundColor: '#020617',
  },
  inputDisabled: {
    backgroundColor: '#111827',
    color: '#64748b',
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 10,
  },
  toggleLabel: {
    fontSize: 14,
    color: '#cbd5e1',
    fontWeight: '600',
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
