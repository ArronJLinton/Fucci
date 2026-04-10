import React, {useState, useEffect, useCallback, useRef, useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  TextInput,
  Image,
  KeyboardAvoidingView,
  Platform,
  ActivityIndicator,
  StatusBar,
  Linking,
} from 'react-native';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import {useQuery} from '@tanstack/react-query';
import type {RootStackParamList, AuthPendingAction} from '../types/navigation';
import type {DebateComment, DebateCard, ReactionCount} from '../types/debate';
import {fetchDebateById} from '../services/debate';
import {
  listComments,
  createComment as apiCreateComment,
  setCommentVote,
  addCommentReaction,
} from '../services/api';
import {useAuth} from '../context/AuthContext';
import {rootNavigateToProfileAuth} from '../navigation/authNavigationActions';
import {rootNavigate} from '../navigation/rootNavigation';
import {AuthGateModal} from '../components/AuthGateModal';

/** Velocity Strike–style debate detail (009) — aligned with MainDebatesScreen */
const BG = '#0B0E14';
const LIME = '#C6FF00';
const CARD = '#1A1F2E';
const TEXT = '#FFFFFF';
const MUTED = '#8B92A5';
const RED = '#FF3B30';
const BORDER_SUB = 'rgba(255,255,255,0.08)';

type SingleDebateRouteProp = RouteProp<RootStackParamList, 'SingleDebate'>;

function formatScore(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

function formatSourcePublishedAt(iso: string): string {
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return '';
    return d.toLocaleDateString(undefined, {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  } catch {
    return '';
  }
}

/**
 * Debate Pulse / feed `binary_consensus` alignment:
 * agree side = upvotes on agree card; disagree side = downvotes on agree + votes on disagree card.
 */
function binaryPulseSideTotals(
  binaryCards: DebateCard[],
  localCounts: Record<number, {upvotes: number; downvotes: number}>,
): {agreeVotes: number; disagreeVotes: number} {
  let agreeVotes = 0;
  let disagreeVotes = 0;
  for (const c of binaryCards) {
    if (c.id == null) continue;
    const counts = localCounts[c.id] ?? {
      upvotes: c.vote_counts?.upvotes ?? 0,
      downvotes: c.vote_counts?.downvotes ?? 0,
    };
    if (c.stance === 'agree') {
      agreeVotes += counts.upvotes;
      disagreeVotes += counts.downvotes;
    } else if (c.stance === 'disagree') {
      disagreeVotes += counts.upvotes + counts.downvotes;
    }
  }
  return {agreeVotes, disagreeVotes};
}

const SingleDebateScreen = () => {
  const route = useRoute<SingleDebateRouteProp>();
  const navigation = useNavigation();
  const insets = useSafeAreaInsets();
  const {match, debate} = route.params;
  const {token, isLoggedIn} = useAuth();

  const debateDetailQuery = useQuery({
    queryKey: ['singleDebate', debate?.id],
    queryFn: async () => {
      if (debate?.id == null) return null;
      return fetchDebateById(debate.id);
    },
    enabled: typeof debate?.id === 'number' && debate.id > 0,
    staleTime: 5 * 60 * 1000,
  });

  const teams = useMemo(() => {
    const payloadTeams = debateDetailQuery.data?.teams ?? debate?.teams;
    if (payloadTeams?.home?.name || payloadTeams?.away?.name) {
      return payloadTeams;
    }
    const matchHomeName = match?.teams?.home?.name?.trim() ?? '';
    const matchAwayName = match?.teams?.away?.name?.trim() ?? '';
    const matchHomeLogo = match?.teams?.home?.logo?.trim() ?? '';
    const matchAwayLogo = match?.teams?.away?.logo?.trim() ?? '';
    if (!matchHomeName && !matchAwayName) {
      return payloadTeams;
    }
    return {
      home: {name: matchHomeName, logo: matchHomeLogo, score: match?.goals?.home ?? undefined},
      away: {name: matchAwayName, logo: matchAwayLogo, score: match?.goals?.away ?? undefined},
    };
  }, [debateDetailQuery.data?.teams, debate?.teams, match]);

  const homeName = teams?.home?.name?.trim() ?? '';
  const awayName = teams?.away?.name?.trim() ?? '';
  const homeLogo = teams?.home?.logo?.trim() ?? '';
  const awayLogo = teams?.away?.logo?.trim() ?? '';
  const homeScore = teams?.home?.score;
  const awayScore = teams?.away?.score;
  const showTeamsRow = !!homeName || !!awayName;
  const showScore =
    debate?.debate_type === 'post_match' &&
    Number.isFinite(homeScore) &&
    Number.isFinite(awayScore);

  const [comments, setComments] = useState<DebateComment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentsError, setCommentsError] = useState<string | null>(null);
  const [commentInput, setCommentInput] = useState('');
  const [replyingToCommentId, setReplyingToCommentId] = useState<number | null>(
    null,
  );
  const [commentSubmitting, setCommentSubmitting] = useState(false);
  const [commentSubmitError, setCommentSubmitError] = useState<string | null>(
    null,
  );
  const [voteLoadingCommentId, setVoteLoadingCommentId] = useState<
    number | null
  >(null);
  const [reactionLoadingCommentId, setReactionLoadingCommentId] = useState<
    number | null
  >(null);
  const [showReactionPickerCommentId, setShowReactionPickerCommentId] =
    useState<number | null>(null);
  /** Comment IDs whose reply threads are collapsed (subcomments hidden) */
  const [collapsedReplyIds, setCollapsedReplyIds] = useState<Set<number>>(
    new Set(),
  );
  /** When non-null, show AuthGateModal; value is the pending action for return-to-debate */
  const [authGatePendingAction, setAuthGatePendingAction] =
    useState<AuthPendingAction | null>(null);

  /** Per-card vote counts for Debate Pulse (from debate payload; vote on this screen is via feed/hero only). */
  const [localCardVoteCounts, setLocalCardVoteCounts] = useState<
    Record<number, {upvotes: number; downvotes: number}>
  >({});

  const allBinaryCards: DebateCard[] = useMemo(
    () =>
      (debate?.cards ?? []).filter(
        c => c.stance === 'agree' || c.stance === 'disagree',
      ),
    [debate?.cards],
  );

  // Initialize per-card counts from debate when debate loads (e.g. new debate)
  useEffect(() => {
    if (!debate?.id || !debate?.cards?.length) return;
    const next: Record<number, {upvotes: number; downvotes: number}> = {};
    debate.cards.forEach(c => {
      if (c.id != null) {
        next[c.id] = {
          upvotes: c.vote_counts?.upvotes ?? 0,
          downvotes: c.vote_counts?.downvotes ?? 0,
        };
      }
    });
    setLocalCardVoteCounts(next);
  }, [debate?.id]);

  const loadComments = useCallback(async () => {
    const debateId = debate?.id;
    if (debateId == null) return;
    setCommentsError(null);
    setCommentsLoading(true);
    try {
      const list = await listComments(debateId);
      setComments(list);
    } catch (_e) {
      setCommentsError('Could not load comments. Tap Retry to try again.');
    } finally {
      setCommentsLoading(false);
    }
  }, [debate?.id]);

  useEffect(() => {
    loadComments();
  }, [loadComments]);

  // After return from Login/SignUp: best-effort auto-initiate pending action (T024)
  const pendingFromAuth = route.params?.pendingAction;
  const consumedPendingRef = useRef(false);
  useEffect(() => {
    if (
      !isLoggedIn ||
      !pendingFromAuth ||
      !match ||
      !debate ||
      consumedPendingRef.current
    )
      return;
    consumedPendingRef.current = true;
    if (pendingFromAuth === 'reply') {
      setReplyingToCommentId(-1);
    } else if (
      pendingFromAuth === 'reaction' &&
      comments.length > 0 &&
      comments[0].id != null
    ) {
      setShowReactionPickerCommentId(comments[0].id);
    }
    (navigation as any).setParams({pendingAction: undefined});
  }, [isLoggedIn, pendingFromAuth, match, debate, comments.length, navigation]);

  const headline = debate?.headline ?? 'Debate';

  const authGateContent = useMemo(
    () => ({
      title: 'Join the conversation',
      message:
        'Sign in or create an account to reply, vote, or react on comments.',
    }),
    [],
  );

  const COMMENT_MAX_LENGTH = 500;
  const DEFAULT_REACTION_EMOJIS = ['👍', '❤️', '🔥', '😂', '👎'];

  const handleAddComment = async () => {
    const content = commentInput.trim();
    if (!content || !debate?.id) return;
    if (content.length > COMMENT_MAX_LENGTH) {
      setCommentSubmitError(
        `Comment must be at most ${COMMENT_MAX_LENGTH} characters`,
      );
      return;
    }
    if (!isLoggedIn || !token) {
      setAuthGatePendingAction('reply');
      return;
    }
    setCommentSubmitError(null);
    const parentIdForApi =
      replyingToCommentId != null && replyingToCommentId > 0
        ? replyingToCommentId
        : undefined;
    setCommentInput('');
    setReplyingToCommentId(null);
    setCommentSubmitting(true);
    try {
      const created = await apiCreateComment(token, debate.id, {
        content,
        parent_comment_id: parentIdForApi,
      });
      if (created) {
        const parentId = created.parent_comment_id ?? null;
        const newComment = {
          ...created,
          reactions: created.reactions ?? [],
          subcomments: created.subcomments ?? [],
        };
        if (parentId != null) {
          setComments(prev =>
            prev.map(c => {
              if (c.id === parentId) {
                return {
                  ...c,
                  subcomments: [...(c.subcomments ?? []), newComment],
                };
              }
              if (c.subcomments?.length) {
                return {
                  ...c,
                  subcomments: c.subcomments.map(sub =>
                    sub.id === parentId
                      ? {
                          ...sub,
                          subcomments: [...(sub.subcomments ?? []), newComment],
                        }
                      : sub,
                  ),
                };
              }
              return c;
            }),
          );
        } else {
          setComments(prev => [...prev, newComment]);
        }
      } else {
        setCommentSubmitError('Failed to post comment. Tap to retry.');
      }
    } catch (_e) {
      setCommentSubmitError('Failed to post comment. Tap to retry.');
    } finally {
      setCommentSubmitting(false);
    }
  };

  const handleCommentVote = async (
    commentId: number,
    newVote: 'upvote' | 'downvote' | null,
  ) => {
    if (!isLoggedIn || !token) {
      setAuthGatePendingAction('vote');
      return;
    }
    const c =
      comments.find(x => x.id === commentId) ??
      [...comments, ...comments.flatMap(x => x.subcomments ?? [])].find(
        x => x.id === commentId,
      );
    const current = c?.current_user_vote ?? null;
    const toSend = current === newVote ? null : newVote;
    setVoteLoadingCommentId(commentId);
    try {
      const res = await setCommentVote(token, commentId, toSend);
      if (res != null) {
        setComments(prev =>
          prev.map(top => {
            if (top.id === commentId) {
              return {
                ...top,
                net_score: res.net_score,
                current_user_vote: toSend ?? undefined,
              };
            }
            if (top.subcomments) {
              return {
                ...top,
                subcomments: top.subcomments.map(sub =>
                  sub.id === commentId
                    ? {
                        ...sub,
                        net_score: res.net_score,
                        current_user_vote: toSend ?? undefined,
                      }
                    : sub,
                ),
              };
            }
            return top;
          }),
        );
      }
    } finally {
      setVoteLoadingCommentId(null);
    }
  };

  const handleCommentReaction = async (commentId: number, emoji: string) => {
    if (!isLoggedIn || !token) {
      setAuthGatePendingAction('reaction');
      return;
    }
    setShowReactionPickerCommentId(null);
    setReactionLoadingCommentId(commentId);
    try {
      const res = await addCommentReaction(token, commentId, emoji);
      if (res != null) {
        setComments(prev =>
          prev.map(top => {
            if (top.id === commentId) return {...top, reactions: res.reactions};
            if (top.subcomments) {
              return {
                ...top,
                subcomments: top.subcomments.map(sub =>
                  sub.id === commentId
                    ? {...sub, reactions: res.reactions}
                    : sub,
                ),
              };
            }
            return top;
          }),
        );
      }
    } finally {
      setReactionLoadingCommentId(null);
    }
  };

  const renderComment = (c: DebateComment, isSub?: boolean) => {
    const voteLoading = voteLoadingCommentId === c.id;
    const reactionLoading = reactionLoadingCommentId === c.id;
    const showPicker = showReactionPickerCommentId === c.id;
    const currentVote = c.current_user_vote ?? null;
    return (
      <View
        key={c.id}
        style={[styles.commentRow, isSub && styles.subcommentRow]}>
        <View style={styles.commentAvatar}>
          {c.user_avatar_url ? (
            <Image
              source={{uri: c.user_avatar_url}}
              style={styles.commentAvatarImage}
            />
          ) : (
            <Text style={styles.commentAvatarText}>
              {(c.user_display_name || '?').charAt(0).toUpperCase()}
            </Text>
          )}
        </View>
        <View style={styles.commentBody}>
          <View style={styles.commentMetaRow}>
            <Text style={styles.commentUsername}>
              {c.user_display_name || 'User'}
            </Text>
            <View style={styles.commentVoteRow}>
              <TouchableOpacity
                onPress={() => handleCommentVote(c.id, 'upvote')}
                disabled={voteLoading}
                style={[
                  styles.commentVoteBtn,
                  currentVote === 'upvote' && styles.commentVoteBtnActive,
                ]}
                hitSlop={{top: 6, bottom: 6, left: 6, right: 6}}>
                {voteLoading ? (
                  <ActivityIndicator size="small" color={MUTED} />
                ) : (
                  <Ionicons
                    name="chevron-up"
                    size={20}
                    color={currentVote === 'upvote' ? LIME : MUTED}
                  />
                )}
              </TouchableOpacity>
              <Text style={styles.commentUpvotes}>
                {c.net_score >= 0 ? '+' : ''}
                {formatScore(c.net_score)}
              </Text>
              <TouchableOpacity
                onPress={() => handleCommentVote(c.id, 'downvote')}
                disabled={voteLoading}
                style={[
                  styles.commentVoteBtn,
                  currentVote === 'downvote' && styles.commentVoteBtnActive,
                ]}
                hitSlop={{top: 6, bottom: 6, left: 6, right: 6}}>
                <Ionicons
                  name="chevron-down"
                  size={20}
                  color={currentVote === 'downvote' ? RED : MUTED}
                />
              </TouchableOpacity>
            </View>
          </View>
          <Text style={styles.commentContent}>{c.content}</Text>
          <View style={styles.commentActionsRow}>
            {!isSub && (
              <TouchableOpacity
                onPress={() => {
                  if (!isLoggedIn) setAuthGatePendingAction('reply');
                  else {
                    setReplyingToCommentId(c.id);
                    setCommentInput('');
                  }
                }}
                style={styles.commentActionBtn}>
                <Ionicons name="arrow-undo-outline" size={14} color={MUTED} />
                <Text style={styles.commentActionText}>Reply</Text>
              </TouchableOpacity>
            )}
            <View style={styles.reactionsRow}>
              {c.reactions?.map((r: ReactionCount, i: number) => (
                <TouchableOpacity
                  key={`${c.id}-r-${i}`}
                  onPress={() => handleCommentReaction(c.id, r.emoji)}
                  disabled={reactionLoading}
                  style={styles.reactionChipTouch}>
                  <Text style={styles.reactionChip}>
                    {r.emoji} {r.count}
                  </Text>
                </TouchableOpacity>
              ))}
              <TouchableOpacity
                onPress={() => {
                  if (!isLoggedIn) setAuthGatePendingAction('reaction');
                  else
                    setShowReactionPickerCommentId(prev =>
                      prev === c.id ? null : c.id,
                    );
                }}
                style={styles.reactionAddBtn}>
                {reactionLoading ? (
                  <ActivityIndicator size="small" color={MUTED} />
                ) : (
                  <Ionicons name="add-circle-outline" size={18} color={MUTED} />
                )}
              </TouchableOpacity>
            </View>
          </View>
          {showPicker && (
            <View style={styles.reactionPickerRow}>
              {DEFAULT_REACTION_EMOJIS.map(emoji => (
                <TouchableOpacity
                  key={emoji}
                  onPress={() => handleCommentReaction(c.id, emoji)}
                  style={styles.reactionPickerEmoji}>
                  <Text style={styles.reactionPickerEmojiText}>{emoji}</Text>
                </TouchableOpacity>
              ))}
            </View>
          )}
          {!isSub && c.subcomments && c.subcomments.length > 0 && (
            <>
              <TouchableOpacity
                onPress={() => {
                  setCollapsedReplyIds(prev => {
                    const next = new Set(prev);
                    if (next.has(c.id)) next.delete(c.id);
                    else next.add(c.id);
                    return next;
                  });
                }}
                style={styles.collapseRepliesRow}
                activeOpacity={0.7}>
                <Ionicons
                  name={
                    collapsedReplyIds.has(c.id) ? 'chevron-down' : 'chevron-up'
                  }
                  size={16}
                  color={MUTED}
                />
                <Text style={styles.collapseRepliesText}>
                  {collapsedReplyIds.has(c.id)
                    ? `Show ${c.subcomments.length} ${c.subcomments.length === 1 ? 'reply' : 'replies'}`
                    : 'Hide replies'}
                </Text>
              </TouchableOpacity>
              {!collapsedReplyIds.has(c.id) &&
                c.subcomments.map(sub => renderComment(sub, true))}
            </>
          )}
          {isSub &&
            c.subcomments &&
            c.subcomments.length > 0 &&
            c.subcomments.map(sub => renderComment(sub, true))}
        </View>
      </View>
    );
  };

  const sourceHeadline = debate?.source_headline?.trim();
  const sourceUrl = debate?.source_url?.trim();
  const sourcePublishedAt = debate?.source_published_at?.trim();

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={0}>
      <StatusBar barStyle="light-content" backgroundColor={BG} />
      <View style={[styles.topBar, {paddingTop: Math.max(insets.top, 8)}]}>
        <TouchableOpacity
          onPress={() => navigation.goBack()}
          style={styles.topBarButton}
          hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}>
          <Ionicons name="chevron-back" size={28} color={LIME} />
        </TouchableOpacity>
      </View>

      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>
        {/* FR-006c: no AI analysis strip; optional source provenance only (FR-009). */}
        {showTeamsRow ? (
          <>
            <View style={styles.teamsRow} accessibilityRole="text">
              <View style={styles.teamSide}>
                {homeLogo ? (
                  <Image source={{uri: homeLogo}} style={styles.teamLogo} resizeMode="contain" />
                ) : (
                  <View style={styles.teamLogoFallback}>
                    <Ionicons name="shield-outline" size={14} color={MUTED} />
                  </View>
                )}
                <Text style={styles.teamName} numberOfLines={2}>
                  {homeName || 'Home'}
                </Text>
              </View>
              <Text style={styles.teamsVs}>
                {showScore ? `${homeScore} - ${awayScore}` : 'VS'}
              </Text>
              <View style={[styles.teamSide, styles.teamSideRight]}>
                {awayLogo ? (
                  <Image source={{uri: awayLogo}} style={styles.teamLogo} resizeMode="contain" />
                ) : (
                  <View style={styles.teamLogoFallback}>
                    <Ionicons name="shield-outline" size={14} color={MUTED} />
                  </View>
                )}
                <Text style={styles.teamName} numberOfLines={2}>
                  {awayName || 'Away'}
                </Text>
              </View>
            </View>
            <View style={styles.teamsHeadlineDivider} />
          </>
        ) : null}
        <Text style={styles.headline}>{headline}</Text>
        {sourceHeadline || sourceUrl || sourcePublishedAt ? (
          <View style={styles.sourceBlock}>
            {sourceHeadline || sourceUrl ? (
              <Text
                style={styles.sourceLine}
                numberOfLines={3}
                onPress={() => {
                  if (!sourceUrl) {
                    return;
                  }
                  try {
                    const parsed = new URL(sourceUrl);
                    const isHttp =
                      parsed.protocol === 'http:' || parsed.protocol === 'https:';
                    if (!isHttp) {
                      return;
                    }
                    Linking.canOpenURL(sourceUrl)
                      .then(supported => {
                        if (supported) {
                          Linking.openURL(sourceUrl).catch(() => {});
                        }
                      })
                      .catch(() => {});
                  } catch {
                    // Invalid URL; do nothing
                  }
                }}>
                {sourceHeadline || sourceUrl}
              </Text>
            ) : null}
            {sourcePublishedAt ? (
              <Text style={styles.sourceMeta}>
                {formatSourcePublishedAt(sourcePublishedAt)}
              </Text>
            ) : null}
          </View>
        ) : null}

        {/* Single binary vote bar (agree / disagree share of community votes) */}
        {(() => {
          if (allBinaryCards.length === 0) return null;
          const agreeCard = allBinaryCards.find(c => c.stance === 'agree');
          const disagreeCard = allBinaryCards.find(c => c.stance === 'disagree');
          const {agreeVotes, disagreeVotes} = binaryPulseSideTotals(
            allBinaryCards,
            localCardVoteCounts,
          );
          const totalSide = agreeVotes + disagreeVotes;
          const agreePct =
            totalSide > 0 ? Math.round((agreeVotes / totalSide) * 100) : 0;
          const disagreePct = totalSide > 0 ? 100 - agreePct : 0;
          const agreeTitle = agreeCard?.title?.trim() || 'Agree';
          const disagreeTitle = disagreeCard?.title?.trim() || 'Disagree';
          const agreeCountLabel =
            agreeVotes === 1 ? '1 vote' : `${agreeVotes} votes`;
          const disagreeCountLabel =
            disagreeVotes === 1 ? '1 vote' : `${disagreeVotes} votes`;

          return (
            <View style={styles.meterSection}>
              <View style={styles.censusBox}>
                <Text style={styles.censusTitle}>DEBATE PULSE</Text>
                <View style={styles.censusPctRow}>
                  <View style={styles.censusPctCol}>
                    <Text style={styles.censusCaption}>AGREE</Text>
                    <Text style={styles.censusPctLarge}>{agreePct}%</Text>
                    <Text style={styles.censusVoteCount}>{agreeCountLabel}</Text>
                  </View>
                  <View style={[styles.censusPctCol, styles.censusPctColEnd]}>
                    <Text style={styles.censusCaption}>DISAGREE</Text>
                    <Text style={styles.censusPctLargeDisagree}>
                      {disagreePct}%
                    </Text>
                    <Text
                      style={[styles.censusVoteCount, styles.censusVoteCountEnd]}>
                      {disagreeCountLabel}
                    </Text>
                  </View>
                </View>
                <View style={styles.censusBarTrack}>
                  <View
                    style={[styles.censusSegAgree, {width: `${agreePct}%`}]}
                  />
                  <View
                    style={[
                      styles.censusSegDisagree,
                      {width: `${disagreePct}%`},
                    ]}
                  />
                </View>
                <View style={styles.censusTitlesRow}>
                  <Text style={styles.censusSideTitleAgree} numberOfLines={2}>
                    {agreeTitle}
                  </Text>
                  <Text
                    style={styles.censusSideTitleDisagree}
                    numberOfLines={2}>
                    {disagreeTitle}
                  </Text>
                </View>
              </View>
            </View>
          );
        })()}

        {/* Comments — API thread (seeded + user); card viewpoints are not shown here via swipe */}
        <View style={styles.commentsHeader}>
          <TouchableOpacity style={styles.sortRow}>
            <Text style={styles.sortLabel}>Top Comments</Text>
            <Ionicons name="chevron-down" size={18} color={MUTED} />
          </TouchableOpacity>
          <TouchableOpacity>
            <Ionicons name="filter" size={20} color={MUTED} />
          </TouchableOpacity>
        </View>

        {/* Comment list — loading, error with Retry, or list (006 US1) */}
        {commentsLoading && (
          <View style={styles.commentsLoading}>
            <ActivityIndicator size="small" color={LIME} />
            <Text style={styles.commentsLoadingText}>Loading comments...</Text>
          </View>
        )}
        {!commentsLoading && commentsError && (
          <View style={styles.commentsError}>
            <Text style={styles.commentsErrorText}>{commentsError}</Text>
            <TouchableOpacity style={styles.retryButton} onPress={loadComments}>
              <Text style={styles.retryButtonText}>Retry</Text>
            </TouchableOpacity>
          </View>
        )}
        {!commentsLoading &&
          !commentsError &&
          comments.length === 0 &&
          debate?.id != null && (
            <Text style={styles.commentsEmpty}>No comments yet.</Text>
          )}
        {!commentsLoading &&
          !commentsError &&
          comments.map(c => renderComment(c))}

        <View style={styles.bottomSpacer} />
      </ScrollView>

      {/* Fixed comment input — T021: guests get read-only thread; signed-in users get composer */}
      {isLoggedIn && commentSubmitError ? (
        <View style={styles.commentSubmitError}>
          <Text style={styles.commentSubmitErrorText}>
            {commentSubmitError}
          </Text>
          <TouchableOpacity onPress={() => setCommentSubmitError(null)}>
            <Text style={styles.commentSubmitErrorDismiss}>Dismiss</Text>
          </TouchableOpacity>
        </View>
      ) : null}
      {isLoggedIn ? (
        <View
          style={[
            styles.inputRow,
            {paddingBottom: Math.max(insets.bottom, 12) + 12},
          ]}>
          <View style={styles.inputAvatar}>
            <Text style={styles.inputAvatarText}>Y</Text>
          </View>
          <View style={styles.inputWrap}>
            {replyingToCommentId != null && (
              <View style={styles.replyHintRow}>
                <Text style={styles.replyHintText}>
                  Replying to{' '}
                  {replyingToCommentId > 0
                    ? (comments.find(x => x.id === replyingToCommentId)
                        ?.user_display_name ?? 'comment')
                    : 'comment'}
                </Text>
                <TouchableOpacity
                  onPress={() => setReplyingToCommentId(null)}
                  hitSlop={{top: 8, bottom: 8, left: 8, right: 8}}>
                  <Ionicons name="close" size={18} color={MUTED} />
                </TouchableOpacity>
              </View>
            )}
            <TextInput
              style={styles.input}
              placeholder={
                replyingToCommentId != null
                  ? 'Write a reply...'
                  : 'Write a comment...'
              }
              placeholderTextColor={MUTED}
              value={commentInput}
              onChangeText={setCommentInput}
              multiline
              maxLength={COMMENT_MAX_LENGTH}
              editable={!commentSubmitting}
            />
            <Text style={styles.inputCharCount}>
              {commentInput.length}/{COMMENT_MAX_LENGTH}
            </Text>
          </View>
          <TouchableOpacity
            onPress={handleAddComment}
            style={[
              styles.sendButton,
              (!commentInput.trim() || commentSubmitting) &&
                styles.sendButtonDisabled,
            ]}
            disabled={!commentInput.trim() || commentSubmitting}>
            {commentSubmitting ? (
              <ActivityIndicator size="small" color={LIME} />
            ) : (
              <Ionicons
                name="send"
                size={20}
                color={commentInput.trim() && !commentSubmitting ? LIME : MUTED}
              />
            )}
          </TouchableOpacity>
        </View>
      ) : (
        <View
          style={[
            styles.guestComposerBar,
            {paddingBottom: Math.max(insets.bottom, 12) + 12},
          ]}>
          <Text style={styles.guestComposerText}>
            Sign in to comment, reply, and vote on the thread.
          </Text>
          <TouchableOpacity
            style={styles.guestSignInBtn}
            onPress={() =>
              rootNavigateToProfileAuth(
                match && debate
                  ? {match, debate, pendingAction: 'reply'}
                  : undefined,
              )
            }
            accessibilityRole="button"
            accessibilityLabel="Sign in to comment">
            <Text style={styles.guestSignInBtnText}>Sign in</Text>
          </TouchableOpacity>
        </View>
      )}

      <AuthGateModal
        visible={authGatePendingAction != null}
        onDismiss={() => setAuthGatePendingAction(null)}
        onLogin={() => {
          const pending = authGatePendingAction;
          setAuthGatePendingAction(null);
          rootNavigateToProfileAuth(
            match && debate
              ? {match, debate, pendingAction: pending ?? undefined}
              : undefined,
          );
        }}
        onSignUp={() => {
          const pending = authGatePendingAction;
          setAuthGatePendingAction(null);
          rootNavigate('SignUp', {
            returnToDebate:
              match && debate
                ? {match, debate, pendingAction: pending ?? undefined}
                : undefined,
          });
        }}
        title={authGateContent.title}
        message={authGateContent.message}
      />
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: BG,
  },
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'flex-start',
    paddingHorizontal: 8,
    paddingTop: 8,
    paddingBottom: 12,
    borderBottomWidth: 1,
    borderBottomColor: BORDER_SUB,
    backgroundColor: BG,
  },
  topBarButton: {
    padding: 4,
    width: 40,
  },
  logoTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: TEXT,
    letterSpacing: 0.5,
  },
  scrollView: {
    flex: 1,
    backgroundColor: BG,
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 24,
  },
  teamsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 10,
    gap: 8,
  },
  teamSide: {
    flex: 1,
    flexDirection: 'column',
    alignItems: 'center',
    gap: 4,
    minWidth: 0,
  },
  teamSideRight: {
    justifyContent: 'center',
  },
  teamLogo: {
    width: 32,
    height: 32,
    borderRadius: 11,
    backgroundColor: 'rgba(255,255,255,0.08)',
  },
  teamLogoFallback: {
    width: 32,
    height: 32,
    borderRadius: 11,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.2)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  teamName: {
    color: TEXT,
    fontSize: 14,
    fontWeight: '800',
    letterSpacing: 0.4,
    textAlign: 'center',
    maxWidth: 120,
  },
  teamsVs: {
    color: TEXT,
    fontSize: 14,
    fontWeight: '800',
    letterSpacing: 0.8,
    minWidth: 54,
    textAlign: 'center',
  },
  teamsHeadlineDivider: {
    alignSelf: 'center',
    width: '80%',
    height: 1,
    backgroundColor: MUTED,
    opacity: 0.6,
    marginBottom: 12,
  },
  meterSection: {
    marginBottom: 20,
  },
  censusBox: {
    backgroundColor: CARD,
    borderRadius: 14,
    padding: 16,
    borderWidth: 1,
    borderColor: BORDER_SUB,
  },
  censusTitle: {
    fontSize: 12,
    fontWeight: '800',
    color: LIME,
    letterSpacing: 1.2,
    marginBottom: 12,
  },
  censusPctRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  censusPctCol: {
    flex: 1,
  },
  censusPctColEnd: {
    alignItems: 'flex-end',
  },
  censusCaption: {
    fontSize: 10,
    fontWeight: '700',
    color: MUTED,
    letterSpacing: 1,
    marginBottom: 4,
  },
  censusPctLarge: {
    fontSize: 26,
    fontWeight: '800',
    color: LIME,
  },
  censusPctLargeDisagree: {
    fontSize: 26,
    fontWeight: '800',
    color: RED,
  },
  censusVoteCount: {
    marginTop: 4,
    fontSize: 12,
    fontWeight: '600',
    color: MUTED,
  },
  censusVoteCountEnd: {
    textAlign: 'right',
  },
  censusBarTrack: {
    flexDirection: 'row',
    height: 12,
    borderRadius: 6,
    overflow: 'hidden',
    backgroundColor: 'rgba(255,255,255,0.06)',
  },
  censusSegAgree: {
    height: '100%',
    backgroundColor: LIME,
  },
  censusSegDisagree: {
    height: '100%',
    backgroundColor: RED,
  },
  censusTitlesRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    gap: 12,
    marginTop: 10,
  },
  censusSideTitleAgree: {
    flex: 1,
    fontSize: 12,
    fontWeight: '700',
    color: LIME,
    lineHeight: 16,
  },
  censusSideTitleDisagree: {
    flex: 1,
    fontSize: 12,
    fontWeight: '700',
    color: RED,
    lineHeight: 16,
    textAlign: 'right',
  },
  meterBar: {
    flexDirection: 'row',
    height: 8,
    borderRadius: 4,
    overflow: 'hidden',
    backgroundColor: 'rgba(255,255,255,0.08)',
  },
  meterFillYes: {
    backgroundColor: LIME,
    minWidth: 0,
  },
  meterFillNo: {
    backgroundColor: RED,
    minWidth: 0,
  },
  meterLabels: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginTop: 6,
    paddingHorizontal: 2,
  },
  meterLabelYes: {
    fontSize: 12,
    color: MUTED,
  },
  meterLabelNo: {
    fontSize: 12,
    color: MUTED,
  },
  headline: {
    fontSize: 22,
    fontWeight: '900',
    fontStyle: 'italic',
    color: TEXT,
    lineHeight: 30,
    marginBottom: 12,
    textAlign: 'center',
    letterSpacing: 0.3,
  },
  sourceBlock: {
    marginBottom: 20,
  },
  sourceLine: {
    fontSize: 13,
    color: LIME,
    opacity: 0.9,
    lineHeight: 18,
    marginBottom: 6,
    textAlign: 'center',
  },
  sourceMeta: {
    fontSize: 11,
    color: MUTED,
    lineHeight: 16,
    textAlign: 'center',
  },
  vsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 20,
  },
  playerBlock: {
    flex: 1,
    alignItems: 'center',
  },
  avatarCircle: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: CARD,
    alignItems: 'center',
    justifyContent: 'center',
    marginBottom: 8,
    overflow: 'hidden',
  },
  avatarImage: {
    width: '100%',
    height: '100%',
  },
  playerName: {
    fontSize: 14,
    fontWeight: '600',
    color: TEXT,
    marginBottom: 8,
    maxWidth: 100,
    textAlign: 'center',
  },
  voteBarBg: {
    height: 6,
    width: '100%',
    maxWidth: 100,
    backgroundColor: 'rgba(255,255,255,0.08)',
    borderRadius: 3,
    overflow: 'hidden',
  },
  voteBarFill: {
    height: '100%',
    borderRadius: 3,
  },
  votePct: {
    fontSize: 13,
    fontWeight: '600',
    color: MUTED,
    marginTop: 4,
  },
  vsBadge: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: CARD,
    alignItems: 'center',
    justifyContent: 'center',
    marginHorizontal: 8,
  },
  vsText: {
    fontSize: 12,
    fontWeight: '800',
    color: TEXT,
  },
  voteNowLinkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    marginTop: 8,
    marginBottom: 28,
    paddingTop: 16,
    borderTopWidth: 1,
    borderTopColor: BORDER_SUB,
  },
  voteNowLinkLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: LIME,
  },
  stanceCardsSection: {
    marginBottom: 28,
  },
  stanceCard: {
    backgroundColor: CARD,
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
    borderColor: BORDER_SUB,
  },
  stanceCardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  stanceCardIcon: {
    fontSize: 24,
  },
  stanceBadge: {
    paddingHorizontal: 12,
    paddingVertical: 4,
    borderRadius: 12,
  },
  stanceBadgeText: {
    fontSize: 12,
    fontWeight: '600',
    color: BG,
    textTransform: 'uppercase',
  },
  stanceCardTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: TEXT,
    marginBottom: 8,
    lineHeight: 22,
  },
  stanceCardDescription: {
    fontSize: 14,
    color: MUTED,
    lineHeight: 20,
    marginBottom: 12,
  },
  stanceVoteBarBg: {
    height: 8,
    backgroundColor: 'rgba(255,255,255,0.08)',
    borderRadius: 4,
    overflow: 'hidden',
  },
  stanceVoteBarFill: {
    height: '100%',
    borderRadius: 4,
  },
  stanceVotePct: {
    fontSize: 12,
    color: MUTED,
    marginTop: 4,
  },
  stanceVoteNowRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    marginTop: 12,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: BORDER_SUB,
  },
  stanceVoteNowLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: LIME,
  },
  commentsHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 16,
  },
  sortRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  sortLabel: {
    fontSize: 14,
    fontWeight: '800',
    color: LIME,
    letterSpacing: 0.8,
  },
  commentRow: {
    flexDirection: 'row',
    marginBottom: 16,
    padding: 12,
    backgroundColor: CARD,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: BORDER_SUB,
  },
  subcommentRow: {
    marginLeft: 12,
    marginBottom: 12,
  },
  commentAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(198,255,0,0.12)',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
    overflow: 'hidden',
  },
  commentAvatarImage: {
    width: 36,
    height: 36,
  },
  commentAvatarText: {
    fontSize: 14,
    fontWeight: '700',
    color: LIME,
  },
  commentBody: {
    flex: 1,
  },
  commentMetaRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
  },
  commentUsername: {
    fontSize: 14,
    fontWeight: '700',
    color: TEXT,
  },
  commentVoteRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  commentVoteBtn: {
    padding: 4,
  },
  commentVoteBtnActive: {},
  commentUpvotes: {
    fontSize: 13,
    fontWeight: '700',
    color: LIME,
    minWidth: 32,
    textAlign: 'center',
  },
  commentContent: {
    fontSize: 14,
    color: MUTED,
    lineHeight: 20,
    marginBottom: 6,
  },
  commentActionsRow: {
    flexDirection: 'row',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: 12,
    marginTop: 6,
  },
  commentActionBtn: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  reactionsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'center',
    gap: 8,
  },
  reactionChipTouch: {
    paddingVertical: 2,
    paddingHorizontal: 4,
  },
  reactionChip: {
    fontSize: 13,
    color: MUTED,
  },
  reactionAddBtn: {
    padding: 4,
  },
  reactionPickerRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginTop: 8,
    paddingVertical: 6,
  },
  reactionPickerEmoji: {
    padding: 6,
    backgroundColor: 'rgba(255,255,255,0.08)',
    borderRadius: 8,
  },
  reactionPickerEmojiText: {
    fontSize: 20,
  },
  collapseRepliesRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
    marginTop: 8,
    paddingVertical: 4,
  },
  collapseRepliesText: {
    fontSize: 13,
    color: LIME,
    fontWeight: '600',
  },
  commentsLoading: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 10,
    paddingVertical: 24,
  },
  commentsLoadingText: {
    fontSize: 14,
    color: MUTED,
  },
  commentsError: {
    paddingVertical: 24,
    alignItems: 'center',
    gap: 12,
  },
  commentsErrorText: {
    fontSize: 14,
    color: MUTED,
    textAlign: 'center',
  },
  retryButton: {
    paddingHorizontal: 20,
    paddingVertical: 10,
    backgroundColor: CARD,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: LIME,
  },
  retryButtonText: {
    fontSize: 14,
    fontWeight: '700',
    color: LIME,
  },
  commentsEmpty: {
    fontSize: 14,
    color: MUTED,
    textAlign: 'center',
    paddingVertical: 24,
  },
  commentActions: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  commentActionItem: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 4,
  },
  commentActionText: {
    fontSize: 12,
    color: LIME,
    fontWeight: '600',
  },
  bottomSpacer: {
    height: 24,
  },
  commentSubmitError: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 10,
    backgroundColor: 'rgba(255,59,48,0.12)',
    borderTopWidth: 1,
    borderTopColor: BORDER_SUB,
  },
  commentSubmitErrorText: {
    fontSize: 13,
    color: RED,
    flex: 1,
  },
  commentSubmitErrorDismiss: {
    fontSize: 13,
    color: MUTED,
    marginLeft: 8,
  },
  guestComposerBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    backgroundColor: BG,
    borderTopWidth: 1,
    borderTopColor: BORDER_SUB,
  },
  guestComposerText: {
    flex: 1,
    fontSize: 13,
    color: MUTED,
    lineHeight: 18,
  },
  guestSignInBtn: {
    backgroundColor: LIME,
    paddingHorizontal: 18,
    paddingVertical: 10,
    borderRadius: 10,
  },
  guestSignInBtnText: {
    fontSize: 14,
    fontWeight: '800',
    color: BG,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: BG,
    borderTopWidth: 1,
    borderTopColor: BORDER_SUB,
    gap: 10,
  },
  inputAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: 'rgba(198,255,0,0.15)',
    alignItems: 'center',
    justifyContent: 'center',
  },
  inputAvatarText: {
    fontSize: 14,
    fontWeight: '700',
    color: LIME,
  },
  inputWrap: {
    flex: 1,
    minWidth: 0,
  },
  replyHintRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 4,
    paddingHorizontal: 4,
  },
  replyHintText: {
    fontSize: 12,
    color: MUTED,
  },
  input: {
    minHeight: 40,
    maxHeight: 100,
    backgroundColor: CARD,
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 15,
    color: TEXT,
    borderWidth: 1,
    borderColor: BORDER_SUB,
  },
  inputCharCount: {
    fontSize: 11,
    color: MUTED,
    marginTop: 2,
    marginLeft: 4,
  },
  sendButton: {
    padding: 8,
    minWidth: 40,
    alignItems: 'center',
    justifyContent: 'center',
  },
  sendButtonDisabled: {
    opacity: 0.5,
  },
});

export default SingleDebateScreen;
