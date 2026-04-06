import React, {useMemo, useState} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Image,
  Alert,
  Dimensions,
} from 'react-native';
import {useNavigation, useRoute} from '@react-navigation/native';
import type {RouteProp} from '@react-navigation/native';
import type {NavigationProp, RootStackParamList} from '../types/navigation';
import {SafeAreaView} from 'react-native-safe-area-context';
import {Ionicons} from '@expo/vector-icons';
import {countryCodeToFlag} from '../data/countries';
import type {ComparePlayerSnapshot} from '../types/comparePlayer';
import {ComparePlayerSearchModal} from '../components/ComparePlayerSearchModal';
import {useAuth} from '../context/AuthContext';
import {listComparePlayerCatalog} from '../services/playerProfile';
import {userFacingApiMessage} from '../services/api';

const {width: SCREEN_W} = Dimensions.get('window');
const CARD_GAP = 10;
const CARD_W = (SCREEN_W - 28 - CARD_GAP) / 2;

const BG = '#0B0E11';
const LIME = '#CCFF00';
const CYAN = '#00E5FF';
const MUTED = '#6b7280';

function statPct(v: number): number {
  return Math.min(100, Math.round((v / 99) * 100));
}

function fmtCountry(p: ComparePlayerSnapshot): string {
  const flag = countryCodeToFlag(p.countryCode);
  const name = p.countryLabel?.trim();
  if (!name && !flag) {
    return '—';
  }
  return [flag, name].filter(Boolean).join(' ').trim() || '—';
}

function fmtClubTeam(p: ComparePlayerSnapshot): string {
  const t = p.team?.trim();
  return t ? t : '—';
}

function fmtPosition(p: ComparePlayerSnapshot): string {
  const a = p.positionAbbrev?.trim();
  return a ? a : '—';
}

function BasicInfoRow({
  label,
  leftText,
  rightText,
}: {
  label: string;
  leftText: string;
  rightText: string;
}) {
  return (
    <View style={styles.basicInfoRow}>
      <Text style={[styles.basicInfoSide, {color: LIME}]} numberOfLines={2}>
        {leftText}
      </Text>
      <Text style={styles.basicInfoCenter}>{label}</Text>
      <Text
        style={[
          styles.basicInfoSide,
          styles.basicInfoSideRight,
          {color: CYAN},
        ]}
        numberOfLines={2}>
        {rightText}
      </Text>
    </View>
  );
}

function DualStatBar({leftVal, rightVal}: {leftVal: number; rightVal: number}) {
  const lp = statPct(leftVal);
  const rp = statPct(rightVal);
  return (
    <View style={styles.dualTrack}>
      <View style={styles.dualHalf}>
        <View style={[styles.dualFillLeft, {width: `${lp}%`}]} />
      </View>
      <View style={[styles.dualHalf, styles.dualHalfRight]}>
        <View style={[styles.dualFillRight, {width: `${rp}%`}]} />
      </View>
    </View>
  );
}

function PlayerCard({
  side,
  player,
  empty,
  onPressEmpty,
}: {
  side: 'left' | 'right';
  player?: ComparePlayerSnapshot;
  empty?: boolean;
  onPressEmpty?: () => void;
}) {
  const accent = side === 'left' ? LIME : CYAN;
  const borderSide =
    side === 'left' ? styles.cardBorderLeft : styles.cardBorderRight;

  if (empty && onPressEmpty) {
    return (
      <TouchableOpacity
        style={[styles.card, styles.cardEmpty, borderSide]}
        onPress={onPressEmpty}
        activeOpacity={0.85}>
        <View style={styles.emptyInner}>
          <Ionicons name="person-add-outline" size={36} color={accent} />
          <Text style={[styles.emptyTitle, {color: accent}]}>
            SELECT PLAYER
          </Text>
          <Text style={styles.emptySub}>COMPARE PERFORMANCE</Text>
        </View>
      </TouchableOpacity>
    );
  }

  if (!player) {
    return null;
  }

  const flag = countryCodeToFlag(player.countryCode);
  return (
    <View style={[styles.card, styles.cardFilled, borderSide]}>
      <View style={styles.cardTopRow}>
        <View style={styles.agePill}>
          <Text style={styles.agePillText}>AGE {player.age ?? '—'}</Text>
        </View>
        <View style={styles.crestCircle}>
          <Ionicons name="shield" size={18} color={mutedHex(accent)} />
        </View>
      </View>
      <View style={styles.cardPhotoWrap}>
        {player.photoUrl ? (
          <Image source={{uri: player.photoUrl}} style={styles.cardPhoto} />
        ) : (
          <View style={styles.cardPhotoPh}>
            <Ionicons name="person" size={44} color="#475569" />
          </View>
        )}
      </View>
      <Text style={[styles.cardName, {color: accent}]} numberOfLines={2}>
        {player.displayName}
      </Text>
      <View style={styles.cardMetaRow}>
        {flag ? <Text style={styles.cardFlag}>{flag}</Text> : null}
        <Text style={styles.cardPos}>{player.positionAbbrev}</Text>
      </View>
      <Text style={styles.cardCountry}>{player.countryLabel}</Text>
    </View>
  );
}

function mutedHex(accent: string): string {
  return accent === LIME ? '#9ca3af' : '#94a3b8';
}

function HeadToHeadCards({
  left,
  right,
}: {
  left: ComparePlayerSnapshot;
  right: ComparePlayerSnapshot;
}) {
  return (
    <View style={styles.h2hRow}>
      <View style={[styles.h2hCard, styles.h2hBorderLeft]}>
        <View style={styles.h2hTop}>
          <Text style={[styles.h2hRating, {color: LIME}]}>{left.rating}</Text>
        </View>
        <Text style={styles.h2hName} numberOfLines={1}>
          {left.displayName}
        </Text>

        <View style={styles.h2hImg}>
          {left.photoUrl ? (
            <Image source={{uri: left.photoUrl}} style={styles.h2hImgInner} />
          ) : (
            <Ionicons name="person" size={48} color="#475569" />
          )}
        </View>
      </View>
      <View style={[styles.h2hCard, styles.h2hBorderRight]}>
        <View style={[styles.h2hTop, styles.h2hTopRev]}>
          <Text style={[styles.h2hRating, {color: CYAN}]}>{right.rating}</Text>
        </View>
        <Text style={styles.h2hName} numberOfLines={1}>
          {right.displayName}
        </Text>
        <View style={styles.h2hImg}>
          {right.photoUrl ? (
            <Image source={{uri: right.photoUrl}} style={styles.h2hImgInner} />
          ) : (
            <Ionicons name="person" size={48} color="#475569" />
          )}
        </View>
      </View>
    </View>
  );
}

export default function PlayerCompareScreen() {
  const navigation = useNavigation<NavigationProp>();
  const route = useRoute<RouteProp<RootStackParamList, 'PlayerCompare'>>();
  const {token} = useAuth();
  const {left} = route.params;

  const [right, setRight] = useState<ComparePlayerSnapshot | null>(null);
  const [searchOpen, setSearchOpen] = useState(false);
  const [catalog, setCatalog] = useState<ComparePlayerSnapshot[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState<string | null>(null);
  const [league] = useState("Ligue 1 McDonald's");
  const [season] = useState('2023 / 2024');

  const excludeIds = useMemo(() => new Set([left.id]), [left.id]);

  const openSearch = async () => {
    setSearchOpen(true);
    if (!token || catalogLoading) return;
    if (catalog.length > 0 && !catalogError) return;
    setCatalogLoading(true);
    setCatalogError(null);
    try {
      const players = await listComparePlayerCatalog(token);
      setCatalog(players);
    } catch (err) {
      setCatalogError(userFacingApiMessage(err));
    } finally {
      setCatalogLoading(false);
    }
  };

  const topStatRows = [
    {key: 'speed', label: 'SPEED', l: left.speed, r: right?.speed},
    {key: 'shooting', label: 'SHOOTING', l: left.shooting, r: right?.shooting},
    {key: 'passing', label: 'PASSING', l: left.passing, r: right?.passing},
    {
      key: 'dribbling',
      label: 'DRIBBLING',
      l: left.dribbling,
      r: right?.dribbling,
    },
    {
      key: 'defending',
      label: 'DEFENDING',
      l: left.defending,
      r: right?.defending,
    },
    {key: 'physical', label: 'PHYSICAL', l: left.physical, r: right?.physical},
    {key: 'stamina', label: 'STAMINA', l: left.stamina, r: right?.stamina},
  ];

  return (
    <SafeAreaView style={styles.container} edges={['top']}>
      <View style={styles.header}>
        <TouchableOpacity
          onPress={() => navigation.goBack()}
          style={styles.headerIcon}
          accessibilityLabel="Go back">
          <Ionicons name="arrow-back" size={24} color={LIME} />
        </TouchableOpacity>
      </View>

      <ScrollView
        style={styles.scroll}
        contentContainerStyle={styles.scrollContent}
        showsVerticalScrollIndicator={false}>
        {!right ? (
          <View style={styles.cardsRow}>
            <PlayerCard side="left" player={left} />
            <PlayerCard
              side="right"
              empty
              onPressEmpty={openSearch}
            />
          </View>
        ) : null}

        {!right ? (
          <>
            <SectionTitle title="TOP STATS" />
            {topStatRows.map(row => (
              <View key={row.key} style={styles.statRow3}>
                <Text style={[styles.statLeft, {color: LIME}]}>{row.l}</Text>
                <Text style={styles.statCenter}>{row.label}</Text>
                <Text style={[styles.statRight, {color: CYAN}]}>
                  {row.r != null ? row.r : '—'}
                </Text>
              </View>
            ))}

            <SectionTitle title="BASIC INFO" />
            <BasicInfoRow
              label="AGE"
              leftText={left.age != null ? String(left.age) : '—'}
              rightText="—"
            />
            <BasicInfoRow
              label="COUNTRY"
              leftText={fmtCountry(left)}
              rightText="—"
            />
            <BasicInfoRow
              label="CLUB TEAM"
              leftText={fmtClubTeam(left)}
              rightText="—"
            />
            <BasicInfoRow
              label="POSITION"
              leftText={fmtPosition(left)}
              rightText="—"
            />
          </>
        ) : (
          <>
            <HeadToHeadCards left={left} right={right} />

            <SectionTitle title="BASIC INFO" />
            <BasicInfoRow
              label="AGE"
              leftText={left.age != null ? String(left.age) : '—'}
              rightText={right.age != null ? String(right.age) : '—'}
            />
            <BasicInfoRow
              label="COUNTRY"
              leftText={fmtCountry(left)}
              rightText={fmtCountry(right)}
            />
            <BasicInfoRow
              label="CLUB TEAM"
              leftText={fmtClubTeam(left)}
              rightText={fmtClubTeam(right)}
            />
            <BasicInfoRow
              label="POSITION"
              leftText={fmtPosition(left)}
              rightText={fmtPosition(right)}
            />

            <Text style={styles.coreCaption}>CORE ATTRIBUTES</Text>
            {(
              [
                {key: 'speed', label: 'SPEED', l: left.speed, r: right.speed},
                {
                  key: 'shooting',
                  label: 'SHOOTING',
                  l: left.shooting,
                  r: right.shooting,
                },
                {
                  key: 'passing',
                  label: 'PASSING',
                  l: left.passing,
                  r: right.passing,
                },
                {
                  key: 'dribbling',
                  label: 'DRIBBLING',
                  l: left.dribbling,
                  r: right.dribbling,
                },
                {
                  key: 'physical',
                  label: 'PHYSICAL',
                  l: left.physical,
                  r: right.physical,
                },
                {
                  key: 'stamina',
                  label: 'STAMINA',
                  l: left.stamina,
                  r: right.stamina,
                },
              ] as const
            ).map(row => (
              <View key={row.key} style={styles.coreBlock}>
                <View style={styles.coreNums}>
                  <Text style={[styles.coreNum, {color: LIME}]}>{row.l}</Text>
                  <Text style={styles.coreLabel}>{row.label}</Text>
                  <Text style={[styles.coreNum, {color: CYAN}]}>{row.r}</Text>
                </View>
                <DualStatBar leftVal={row.l} rightVal={row.r} />
              </View>
            ))}

            <TouchableOpacity
              style={styles.changeOpp}
              onPress={openSearch}>
              <Text style={styles.changeOppText}>Change opponent</Text>
            </TouchableOpacity>
          </>
        )}
      </ScrollView>

      <ComparePlayerSearchModal
        visible={searchOpen}
        players={catalog}
        loading={catalogLoading}
        loadError={catalogError}
        onClose={() => setSearchOpen(false)}
        excludeIds={excludeIds}
        onSelect={p => setRight(p)}
      />
    </SafeAreaView>
  );
}

function SectionTitle({title}: {title: string}) {
  return (
    <View style={styles.sectionTitleRow}>
      <View style={styles.sectionHair} />
      <Text style={styles.sectionTitleText}>{title}</Text>
      <View style={styles.sectionHair} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {flex: 1, backgroundColor: BG},
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 8,
    paddingVertical: 10,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1e293b',
  },
  headerIcon: {
    width: 44,
    height: 44,
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 14,
    fontWeight: '900',
    color: LIME,
    letterSpacing: 1,
  },
  scroll: {flex: 1},
  scrollContent: {paddingBottom: 40},
  cardsRow: {
    flexDirection: 'row',
    paddingHorizontal: 14,
    gap: CARD_GAP,
    marginTop: 12,
  },
  card: {
    width: CARD_W,
    minHeight: 220,
    borderRadius: 12,
    backgroundColor: '#0f1419',
    borderWidth: 1,
    borderColor: '#1e293b',
    padding: 10,
  },
  cardFilled: {},
  cardEmpty: {
    borderStyle: 'dashed',
    justifyContent: 'center',
  },
  cardBorderLeft: {borderBottomWidth: 3, borderBottomColor: LIME},
  cardBorderRight: {borderBottomWidth: 3, borderBottomColor: CYAN},
  cardTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  agePill: {
    backgroundColor: '#1e293b',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
  },
  agePillText: {fontSize: 10, fontWeight: '800', color: '#e2e8f0'},
  crestCircle: {
    width: 28,
    height: 28,
    borderRadius: 14,
    backgroundColor: '#1e293b',
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardPhotoWrap: {
    alignItems: 'center',
    marginBottom: 8,
  },
  cardPhoto: {width: 72, height: 72, borderRadius: 36},
  cardPhotoPh: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: '#1e293b',
    alignItems: 'center',
    justifyContent: 'center',
  },
  cardName: {
    fontSize: 13,
    fontWeight: '900',
    letterSpacing: 0.5,
    marginBottom: 4,
  },
  cardMetaRow: {flexDirection: 'row', alignItems: 'center', gap: 6},
  cardFlag: {fontSize: 14},
  cardPos: {
    fontSize: 12,
    fontWeight: '800',
    color: CYAN,
  },
  cardCountry: {
    fontSize: 10,
    fontWeight: '600',
    color: MUTED,
    marginTop: 4,
  },
  emptyInner: {alignItems: 'center', paddingVertical: 16},
  emptyTitle: {
    marginTop: 12,
    fontSize: 11,
    fontWeight: '900',
    letterSpacing: 0.8,
  },
  emptySub: {
    marginTop: 6,
    fontSize: 9,
    fontWeight: '600',
    color: MUTED,
    letterSpacing: 0.5,
  },
  filtersRow: {
    flexDirection: 'row',
    gap: 10,
    paddingHorizontal: 14,
    marginTop: 16,
  },
  filterChip: {
    flex: 1,
    backgroundColor: '#111827',
    borderRadius: 10,
    padding: 12,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  filterLabel: {
    fontSize: 9,
    fontWeight: '800',
    color: MUTED,
    letterSpacing: 0.6,
    marginBottom: 4,
  },
  filterValue: {
    fontSize: 11,
    fontWeight: '700',
    color: '#e2e8f0',
    marginBottom: 4,
  },
  sectionTitleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginTop: 28,
    marginBottom: 14,
    paddingHorizontal: 14,
    gap: 12,
  },
  sectionHair: {flex: 1, height: 1, backgroundColor: '#1e293b'},
  sectionTitleText: {
    fontSize: 11,
    fontWeight: '800',
    color: MUTED,
    letterSpacing: 1.2,
  },
  statRow3: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    paddingHorizontal: 14,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1a222d',
  },
  basicInfoRow: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 10,
    paddingHorizontal: 14,
    gap: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: '#1a222d',
  },
  basicInfoSide: {
    flex: 1,
    minWidth: 0,
    fontSize: 12,
    fontWeight: '800',
    textAlign: 'left',
  },
  basicInfoSideRight: {
    textAlign: 'right',
  },
  basicInfoCenter: {
    width: 92,
    flexShrink: 0,
    textAlign: 'center',
    fontSize: 10,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 0.5,
  },
  statLeft: {
    width: 52,
    fontSize: 14,
    fontWeight: '800',
    textAlign: 'left',
  },
  statCenter: {
    flex: 1,
    textAlign: 'center',
    fontSize: 11,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 0.8,
  },
  statRight: {
    width: 52,
    fontSize: 14,
    fontWeight: '800',
    textAlign: 'right',
  },
  badgeRow: {alignItems: 'center', marginTop: 20},
  badgeTiny: {
    fontSize: 10,
    fontWeight: '800',
    color: CYAN,
    letterSpacing: 1.5,
    backgroundColor: '#111827',
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 6,
    overflow: 'hidden',
  },
  heroTitle: {
    textAlign: 'center',
    fontSize: 26,
    fontWeight: '900',
    marginTop: 12,
    marginBottom: 16,
    letterSpacing: 0.5,
  },
  h2hRow: {
    flexDirection: 'row',
    gap: CARD_GAP,
    paddingHorizontal: 14,
  },
  h2hCard: {
    flex: 1,
    backgroundColor: '#0f1419',
    borderRadius: 12,
    padding: 12,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  h2hBorderLeft: {borderBottomWidth: 3, borderBottomColor: LIME},
  h2hBorderRight: {borderBottomWidth: 3, borderBottomColor: CYAN},
  h2hTop: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
  },
  h2hTopRev: {flexDirection: 'row'},
  h2hRating: {fontSize: 36, fontWeight: '900', lineHeight: 40},
  posBox: {
    backgroundColor: '#1e293b',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
  },
  posBoxText: {fontSize: 11, fontWeight: '800', color: '#e2e8f0'},
  h2hName: {
    fontSize: 13,
    fontWeight: '900',
    color: '#f8fafc',
    marginTop: 8,
    letterSpacing: 0.4,
  },
  h2hTeam: {
    fontSize: 10,
    fontWeight: '600',
    color: MUTED,
    marginTop: 4,
    marginBottom: 10,
  },
  h2hImg: {
    height: 100,
    borderRadius: 8,
    backgroundColor: '#1a222d',
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  h2hImgInner: {width: '100%', height: '100%'},
  coreCaption: {
    textAlign: 'center',
    fontSize: 10,
    fontWeight: '800',
    color: MUTED,
    letterSpacing: 2,
    marginTop: 28,
    marginBottom: 12,
  },
  coreBlock: {paddingHorizontal: 14, marginBottom: 14},
  coreNums: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 6,
  },
  coreNum: {fontSize: 14, fontWeight: '800', minWidth: 36},
  coreLabel: {
    flex: 1,
    textAlign: 'center',
    fontSize: 10,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 0.8,
  },
  dualTrack: {
    flexDirection: 'row',
    height: 8,
    borderRadius: 4,
    overflow: 'hidden',
    backgroundColor: '#1f2937',
  },
  dualHalf: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'stretch',
  },
  dualHalfRight: {justifyContent: 'flex-end'},
  dualFillLeft: {
    height: '100%',
    backgroundColor: LIME,
    borderTopLeftRadius: 4,
    borderBottomLeftRadius: 4,
  },
  dualFillRight: {
    height: '100%',
    backgroundColor: CYAN,
    borderTopRightRadius: 4,
    borderBottomRightRadius: 4,
  },
  valueRow: {
    flexDirection: 'row',
    gap: CARD_GAP,
    paddingHorizontal: 14,
    marginTop: 20,
  },
  valueCol: {
    flex: 1,
    backgroundColor: '#0f1419',
    borderRadius: 10,
    padding: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  valueLbl: {
    fontSize: 10,
    fontWeight: '700',
    color: MUTED,
    marginBottom: 6,
    letterSpacing: 0.8,
  },
  valueAmt: {fontSize: 22, fontWeight: '900'},
  seasonBadge: {alignItems: 'center', marginTop: 24},
  seasonBadgeText: {
    fontSize: 11,
    fontWeight: '800',
    color: '#e2e8f0',
    backgroundColor: '#1f2937',
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 6,
    overflow: 'hidden',
  },
  goalsRow: {
    flexDirection: 'row',
    marginTop: 16,
    paddingHorizontal: 14,
    gap: CARD_GAP,
  },
  goalCol: {
    flex: 1,
    alignItems: 'center',
    gap: 8,
    paddingVertical: 12,
    backgroundColor: '#0f1419',
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#1e293b',
  },
  goalNum: {
    fontSize: 14,
    fontWeight: '800',
    color: '#f8fafc',
    letterSpacing: 0.5,
  },
  changeOpp: {
    marginTop: 24,
    alignSelf: 'center',
    paddingVertical: 10,
    paddingHorizontal: 20,
  },
  changeOppText: {
    fontSize: 13,
    fontWeight: '700',
    color: LIME,
    textDecorationLine: 'underline',
  },
});
