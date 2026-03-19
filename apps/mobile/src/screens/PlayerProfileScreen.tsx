import React, {useState, useEffect, useCallback, useRef} from 'react';
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
  NativeSyntheticEvent,
  NativeScrollEvent,
  Dimensions,
} from 'react-native';
import {useNavigation, useFocusEffect} from '@react-navigation/native';
import type {NavigationProp} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {countryCodeToFlag, COUNTRIES} from '../data/countries';
import {
  getPlayerProfile,
  updatePlayerProfile,
  deletePlayerProfile,
  setPlayerProfileTraits,
} from '../services/playerProfile';
import type {PlayerProfile as PlayerProfileType} from '../types/playerProfile';
import {CountryPicker} from '../components/CountryPicker';
import {PlayerTraitsModal} from '../components/PlayerTraitsModal';

type TabId = 'profile' | 'stats' | 'career';

/** Trait code to Ionicon name for gamified hexagonal badges (FIFA-style) */
const TRAIT_ICONS: Record<string, string> = {
  LEADERSHIP: 'star',
  FINESSE_SHOT: 'football',
  PLAYMAKER: 'git-branch',
  SPEED_DRIBBLER: 'flash',
  LONG_SHOT_TAKER: 'arrow-redo',
  OUTSIDE_FOOT_SHOT: 'footsteps',
  POWER_HEADER: 'ellipse',
  FLAIR: 'ribbon',
  POWER_FREE_KICK: 'lock-closed',
};

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

const {width: SCREEN_WIDTH, height: SCREEN_HEIGHT} = Dimensions.get('window');
const HERO_HEIGHT = Math.max(320, SCREEN_HEIGHT * 0.45);

export default function PlayerProfileScreen() {
  const navigation = useNavigation<NavigationProp>();
  const {token, user} = useAuth();
  const [profile, setProfile] = useState<PlayerProfileType | null>(null);
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
  const [editMode, setEditMode] = useState(false);
  const [showPositionPicker, setShowPositionPicker] = useState(false);
  const traitsScrollRef = useRef<ScrollView>(null);
  const [traitsPageIndex, setTraitsPageIndex] = useState(0);

  const POSITIONS: {value: PlayerProfileType['position']; label: string}[] = [
    {value: 'GK', label: 'Goalkeeper'},
    {value: 'DEF', label: 'Defender'},
    {value: 'MID', label: 'Midfielder'},
    {value: 'FWD', label: 'Forward'},
  ];

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
        setEditAge(p.age != null ? String(p.age) : '');
        setEditCountryCode(p.country || null);
        const countryName = p.country
          ? (COUNTRIES.find(c => c.code === p.country)?.name ?? p.country)
          : '';
        setEditCountryName(countryName);
        setEditClub(p.club ?? '');
        setEditIsFreeAgent(p.is_free_agent ?? false);
        setEditPosition(p.position ?? null);
      } else {
        setProfile(null);
        navigation.replace('CreatePlayerProfile');
      }
    } catch {
      setError('Failed to load profile.');
    } finally {
      setLoading(false);
    }
  }, [token, navigation]);

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
    if (!token || !profile) return;
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
      const updated = await updatePlayerProfile(token, {
        country: editCountryCode,
        position: editPosition,
        age: ageNum,
        club: editIsFreeAgent ? null : editClub.trim() || null,
        is_free_agent: editIsFreeAgent,
      });
      if (updated) {
        setProfile(updated);
        setEditMode(false);
      } else {
        setSaveError('Failed to save. Try again.');
      }
    } catch {
      setSaveError('Failed to save. Try again.');
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
            const ok = await deletePlayerProfile(token);
            if (ok) {
              setProfile(null);
              navigation.replace('CreatePlayerProfile');
            } else {
              setError('Failed to delete profile.');
            }
          },
        },
      ],
    );
  };

  const handleSaveTraits = async (traits: string[]) => {
    if (!token) return;
    const updated = await setPlayerProfileTraits(token, traits);
    if (updated && profile) {
      setProfile({...profile, traits: updated});
    }
    setShowTraitsModal(false);
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
            <Ionicons name="arrow-back" size={24} color="#000" />
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

  const traits = profile.traits ?? [];
  const traitsNumPages = Math.max(1, Math.ceil(traits.length / 3));
  const onTraitsScroll = (e: NativeSyntheticEvent<NativeScrollEvent>) => {
    const x = e.nativeEvent.contentOffset.x;
    const page = Math.round(x / (SCREEN_WIDTH - 48)) || 0;
    setTraitsPageIndex(Math.min(page, traitsNumPages - 1));
  };
  const flag =
    editCountryCode || profile.country
      ? countryCodeToFlag(editCountryCode || profile.country)
      : '';
  const countryName =
    COUNTRIES.find(c => c.code === profile.country)?.name ?? profile.country;

  const posAbbrev =
    profile.position === 'GK'
      ? 'GK'
      : profile.position === 'DEF'
        ? 'CB'
        : profile.position === 'MID'
          ? 'CM'
          : 'ST';

  // Light-weight “card rating” for now (until we have real attributes).
  const seed = (profile.id ?? 1) * 2654435761;
  const stat = (i: number, min: number, max: number) => {
    const x = Math.abs(Math.sin(seed + i * 97.13));
    return Math.round(min + x * (max - min));
  };
  const stats = [
    stat(1, 60, 95),
    stat(2, 55, 92),
    stat(3, 50, 90),
    stat(4, 55, 95),
    stat(5, 35, 88),
    stat(6, 45, 92),
  ];
  const overall = Math.round(stats.reduce((s, v) => s + v, 0) / stats.length);
  const overallClamped = Math.max(1, Math.min(99, overall));

  const clubKey = (profile.club ?? '').toLowerCase();
  const theme = profile.is_free_agent
    ? {a: '#f59e0b', b: '#16a34a'} // gold/green “special”
    : clubKey.includes('juventus')
      ? {a: '#111827', b: '#f59e0b'}
      : clubKey.includes('manchester')
        ? {a: '#b91c1c', b: '#111827'}
        : clubKey.includes('barcelona')
          ? {a: '#7c3aed', b: '#b91c1c'}
          : {a: '#16a34a', b: '#0ea5e9'};

  const selectedTraitsSet = new Set(profile.traits ?? []);
  const archetype = selectedTraitsSet.has('SPEED_DRIBBLER')
    ? 'Speed Demon'
    : selectedTraitsSet.has('LEADERSHIP')
      ? 'Leader'
      : selectedTraitsSet.has('PLAYMAKER')
        ? 'Playmaker'
        : selectedTraitsSet.has('LONG_SHOT_TAKER')
          ? 'Sniper'
          : profile.is_free_agent
            ? 'Free Agent'
            : 'Rising Star';

  const clubOrStatus = profile.is_free_agent ? 'Free Agent' : (profile.club ?? 'Club');
  const metaInfo = [flag ? `${flag} ${countryName}` : countryName, clubOrStatus, posAbbrev].filter(Boolean).join(' • ');

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.header}>
        <TouchableOpacity
          onPress={() => navigation.goBack()}
          style={styles.backBtn}>
          <Ionicons name="arrow-back" size={24} color="#000" />
        </TouchableOpacity>
        <View style={styles.headerIconWrap}>
          <Ionicons name="person-circle-outline" size={26} color="#374151" />
          <View style={styles.headerBadge} />
        </View>
      </View>

      <View style={styles.tabs}>
        {(['profile', 'stats', 'career'] as const).map(tab => (
          <TouchableOpacity
            key={tab}
            style={[styles.tab, activeTab === tab && styles.tabActive]}
            onPress={() => setActiveTab(tab)}>
            <Text
              style={[
                styles.tabText,
                activeTab === tab && styles.tabTextActive,
              ]}>
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
            </Text>
          </TouchableOpacity>
        ))}
      </View>

      {activeTab === 'profile' && (
        <ScrollView
          style={styles.scroll}
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}>
          {/* HERO SECTION (Top ~45%) — Figma layers */}
          <View style={[styles.heroSection, {height: HERO_HEIGHT}]}>
            {/* bg_gradient — gradient / stadium background */}
            <View style={[styles.bg_gradient, {backgroundColor: theme.a}]} />
            <View style={[styles.bg_gradient_bottom, {backgroundColor: theme.b}]} />

            {/* bg_particles — optional overlay */}
            <View style={styles.bg_particles} pointerEvents="none">
              {Array.from({length: 24}, (_, i) => (
                <View
                  key={i}
                  style={[
                    styles.particle,
                    {
                      left: `${(i * 17) % 100}%`,
                      top: `${(i * 13) % 100}%`,
                    },
                  ]}
                />
              ))}
            </View>

            {/* player_image_cutout */}
            <View style={styles.player_image_cutout}>
              <Image
                source={require('./player_profile_action.png')}
                style={styles.playerCutoutImage}
                resizeMode="contain"
              />
            </View>

            {/* Bottom strip: rating_badge, player_name, meta_info, archetype_label */}
            <View style={styles.heroInfoStrip}>
              <View style={styles.heroNameRow}>
                <Text style={styles.player_name} numberOfLines={1}>
                  {user?.display_name ?? 'Player'}
                </Text>
                <View style={styles.rating_badge}>
                  <Text style={styles.ratingValue}>{overallClamped}</Text>
                  <Ionicons name="star" size={14} color="#fbbf24" />
                </View>
              </View>
              <Text style={styles.meta_info} numberOfLines={1}>
                {metaInfo}
              </Text>
              <Text style={styles.archetype_label} numberOfLines={1}>
                {archetype}
              </Text>
            </View>
          </View>

          {editMode ? (
            <View style={styles.editFormCard}>
              <View style={styles.editForm}>
                <Text style={styles.label}>Age</Text>
                <TextInput
                  style={styles.input}
                  value={editAge}
                  onChangeText={setEditAge}
                  placeholder="13–60"
                  placeholderTextColor="#999"
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
                  <Ionicons name="chevron-forward" size={18} color="#999" />
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
                  placeholderTextColor="#999"
                  editable={!editIsFreeAgent && !saving}
                />
                <View style={styles.toggleRow}>
                  <Text style={styles.toggleLabel}>Free Agent</Text>
                  <Switch
                    value={editIsFreeAgent}
                    onValueChange={setEditIsFreeAgent}
                    trackColor={{false: '#ddd', true: '#22c55e'}}
                    thumbColor="#fff"
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
                      ? (POSITIONS.find(p => p.value === editPosition)?.label ??
                        editPosition)
                      : 'Select'}
                  </Text>
                  <Ionicons name="chevron-forward" size={18} color="#999" />
                </View>
              </TouchableOpacity>
            </View>
          ) : null}

          {/* Traits as rarity-based badges (perks) */}
          <View style={styles.traitsSection}>
            <View style={styles.traitsHeader}>
              <Text style={styles.sectionTitle}>Player Traits</Text>
              <TouchableOpacity
                style={styles.addTraitsBtn}
                onPress={() => setShowTraitsModal(true)}>
                <Ionicons name="add-circle-outline" size={20} color="#16a34a" />
                <Text style={styles.addTraitsText}>Add Traits</Text>
              </TouchableOpacity>
            </View>
            <View style={styles.traitsBadgesRow}>
              {(profile.traits ?? []).length === 0 ? (
                <View style={styles.traitsEmptyCard}>
                  <Text style={styles.traitsEmpty}>No traits yet</Text>
                  <Text style={styles.traitsEmptyHint}>Tap "Add Traits" to choose up to 5</Text>
                </View>
              ) : (
                (profile.traits ?? []).map((code: string) => {
                  const rarity =
                    code === 'LEADERSHIP' || code === 'FLAIR'
                      ? 'legendary'
                      : code === 'LONG_SHOT_TAKER' || code === 'POWER_FREE_KICK'
                        ? 'elite'
                        : code === 'SPEED_DRIBBLER' || code === 'POWER_HEADER'
                          ? 'rare'
                          : 'common';
                  const badgeStyle =
                    rarity === 'legendary'
                      ? styles.traitBadgeLegendary
                      : rarity === 'elite'
                        ? styles.traitBadgeElite
                        : rarity === 'rare'
                          ? styles.traitBadgeRare
                          : styles.traitBadgeCommon;

                  return (
                    <View key={code} style={styles.traitBadgeShell}>
                      <View style={[styles.traitBadgeHex, badgeStyle]}>
                        <Ionicons
                          name={(TRAIT_ICONS[code] ?? 'shield-outline') as any}
                          size={22}
                          color="#fff"
                        />
                      </View>
                      <Text
                        style={styles.traitBadgeLabel}
                        numberOfLines={2}>
                        {TRAIT_LABELS[code] ?? code.replace(/_/g, ' ')}
                      </Text>
                    </View>
                  );
                })
              )}
            </View>
          </View>

          {/* Gamified: Career Teams list */}
          <View style={styles.careerSection}>
            <Text style={styles.sectionTitle}>Career Teams</Text>
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
                      <Text style={styles.careerTeamName}>{ct.team_name}</Text>
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

          {editMode && (
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
          )}

          {!editMode && (
            <>
              <TouchableOpacity style={[styles.ctaBtn, styles.ctaBtnPrimary]}>
                <Text style={styles.ctaBtnText}>VIEW PROFILE</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={styles.editLink}
                onPress={() => setEditMode(true)}>
                <Text style={styles.editLinkText}>Edit Profile</Text>
              </TouchableOpacity>
            </>
          )}

          <TouchableOpacity
            style={styles.deleteBtn}
            onPress={handleDeleteProfile}>
            <Text style={styles.deleteBtnText}>Delete Player Profile</Text>
          </TouchableOpacity>
        </ScrollView>
      )}

      {activeTab === 'stats' && (
        <View style={styles.placeholder}>
          <Ionicons name="stats-chart-outline" size={48} color="#ccc" />
          <Text style={styles.placeholderText}>Stats coming soon</Text>
        </View>
      )}

      {activeTab === 'career' && (
        <View style={styles.placeholder}>
          <Ionicons name="list-outline" size={48} color="#ccc" />
          <Text style={styles.placeholderText}>
            Career teams coming in Phase 6
          </Text>
        </View>
      )}

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
    backgroundColor: '#f0f2f5',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#e5e7eb',
  },
  backBtn: {padding: 4},
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
    borderColor: '#fff',
  },
  headerSpacer: {width: 32},
  tabs: {
    flexDirection: 'row',
    backgroundColor: '#fff',
    borderBottomWidth: 1,
    borderBottomColor: '#e5e7eb',
  },
  tab: {
    flex: 1,
    paddingVertical: 14,
    alignItems: 'center',
  },
  tabActive: {
    borderBottomWidth: 3,
    borderBottomColor: '#16a34a',
  },
  tabText: {fontSize: 15, color: '#6b7280', fontWeight: '500'},
  tabTextActive: {fontSize: 15, fontWeight: '700', color: '#16a34a'},
  scroll: {flex: 1},
  scrollContent: {padding: 20, paddingBottom: 48},
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
  /* --- HERO SECTION (Figma layers) --- */
  heroSection: {
    width: '100%',
    marginBottom: 18,
    borderRadius: 22,
    overflow: 'hidden',
    position: 'relative',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 8},
    shadowOpacity: 0.12,
    shadowRadius: 16,
    elevation: 6,
  },
  bg_gradient: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    height: '60%',
    opacity: 0.92,
  },
  bg_gradient_bottom: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    height: '50%',
    opacity: 0.88,
  },
  bg_particles: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
    opacity: 0.12,
  },
  particle: {
    position: 'absolute',
    width: 4,
    height: 4,
    borderRadius: 2,
    backgroundColor: 'rgba(255,255,255,0.8)',
  },
  player_image_cutout: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
    justifyContent: 'center',
    alignItems: 'center',
  },
  playerCutoutImage: {
    width: '100%',
    height: '100%',
  },
  heroInfoStrip: {
    position: 'absolute',
    left: 0,
    right: 0,
    bottom: 0,
    paddingHorizontal: 18,
    paddingVertical: 16,
    paddingTop: 24,
    backgroundColor: 'rgba(0,0,0,0.5)',
  },
  heroNameRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
  },
  player_name: {
    fontSize: 22,
    fontWeight: '800',
    color: '#fff',
    flex: 1,
  },
  rating_badge: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    backgroundColor: 'rgba(255,255,255,0.2)',
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 12,
  },
  ratingValue: {
    fontSize: 16,
    fontWeight: '800',
    color: '#fff',
  },
  meta_info: {
    fontSize: 13,
    fontWeight: '600',
    color: 'rgba(255,255,255,0.9)',
    marginBottom: 2,
  },
  archetype_label: {
    fontSize: 12,
    fontWeight: '600',
    color: 'rgba(255,255,255,0.75)',
  },
  editForm: {marginBottom: 16},
  editFormRow: {marginBottom: 16},
  field: {marginBottom: 16},
  label: {fontSize: 12, color: '#6b7280', marginBottom: 4, fontWeight: '600'},
  value: {fontSize: 16, color: '#000'},
  hint: {fontSize: 12, color: '#9ca3af', marginTop: 2},
  row: {marginBottom: 16},
  rowValue: {flexDirection: 'row', alignItems: 'center'},
  flag: {fontSize: 20, marginRight: 8},
  traitsSection: {
    marginBottom: 24,
    backgroundColor: '#fff',
    borderRadius: 16,
    paddingVertical: 14,
    paddingHorizontal: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 3,
  },
  traitsHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 14,
  },
  sectionTitle: {fontSize: 17, fontWeight: '700', color: '#111827'},
  addTraitsBtn: {flexDirection: 'row', alignItems: 'center', gap: 6},
  addTraitsText: {fontSize: 14, color: '#16a34a', fontWeight: '600'},
  traitsBadgesRow: {
    marginTop: 10,
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    rowGap: 10,
  },
  traitBadgeShell: {
    width: (SCREEN_WIDTH - 16 * 2 - 10 * 2) / 3,
    maxWidth: 110,
    alignItems: 'center',
  },
  traitBadgeHex: {
    width: 64,
    height: 64,
    borderRadius: 18,
    borderWidth: 2,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 6,
  },
  traitBadgeCommon: {
    backgroundColor: '#f9fafb',
    borderColor: '#e5e7eb',
  },
  traitBadgeRare: {
    backgroundColor: '#e0f2fe',
    borderColor: '#38bdf8',
  },
  traitBadgeElite: {
    backgroundColor: '#ede9fe',
    borderColor: '#a855f7',
  },
  traitBadgeLegendary: {
    backgroundColor: '#fef3c7',
    borderColor: '#f59e0b',
  },
  traitBadgeLocked: {
    opacity: 0.55,
  },
  traitBadgeLabel: {
    fontSize: 11,
    fontWeight: '700',
    color: '#111827',
    textAlign: 'center',
  },
  traitBadgeLabelLocked: {
    color: '#9ca3af',
  },
  traitsEmptyCard: {
    paddingVertical: 24,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 12,
    backgroundColor: '#f9fafb',
    borderWidth: 1,
    borderColor: '#e5e7eb',
    borderStyle: 'dashed',
  },
  traitsEmpty: {fontSize: 14, color: '#9ca3af'},
  traitsEmptyHint: {fontSize: 12, color: '#9ca3af', marginTop: 4},
  careerSection: {
    marginBottom: 24,
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 16,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 3,
  },
  careerRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#f3f4f6',
  },
  careerRowLast: {
    borderBottomWidth: 0,
  },
  editFormCard: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 20,
    marginBottom: 20,
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 2},
    shadowOpacity: 0.06,
    shadowRadius: 8,
    elevation: 3,
  },
  careerLogo: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#f3f4f6',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 14,
  },
  careerInfo: {flex: 1},
  careerTeamName: {fontSize: 16, fontWeight: '600', color: '#111827'},
  careerTenure: {fontSize: 13, color: '#6b7280', marginTop: 2},
  careerEmpty: {fontSize: 14, color: '#9ca3af', paddingVertical: 12},
  saveError: {color: '#dc2626', fontSize: 14, marginBottom: 8},
  ctaBtn: {
    borderRadius: 14,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  ctaBtnPrimary: {
    backgroundColor: '#16a34a',
    shadowColor: '#15803d',
    shadowOffset: {width: 0, height: 4},
    shadowOpacity: 0.3,
    shadowRadius: 6,
    elevation: 4,
  },
  ctaBtnText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
    letterSpacing: 0.5,
  },
  editLink: {alignItems: 'center', marginTop: 14},
  editLinkText: {fontSize: 15, color: '#16a34a', fontWeight: '600'},
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
  cancelEditText: {fontSize: 15, color: '#6b7280'},
  deleteBtn: {alignItems: 'center', marginTop: 24},
  deleteBtnText: {fontSize: 14, color: '#dc2626'},
  placeholder: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  placeholderText: {marginTop: 12, color: '#9ca3af'},
  input: {
    borderWidth: 1,
    borderColor: '#e5e7eb',
    borderRadius: 10,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 16,
    color: '#000',
    backgroundColor: '#fff',
  },
  inputDisabled: {
    backgroundColor: '#f5f5f5',
    color: '#999',
  },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 10,
  },
  toggleLabel: {
    fontSize: 15,
    color: '#374151',
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
