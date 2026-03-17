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
  PanResponder,
  Dimensions,
} from 'react-native';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import type {RootStackParamList, AuthPendingAction} from '../types/navigation';
import type {
  DebateComment,
  DebateCard,
  CardVoteTotals,
  ReactionCount,
} from '../types/debate';
import {
  listComments,
  setCardVote,
  createComment as apiCreateComment,
  setCommentVote,
  addCommentReaction,
} from '../services/api';
import {useAuth} from '../context/AuthContext';
import {rootNavigate} from '../navigation/rootNavigation';
import {AuthGateModal} from '../components/AuthGateModal';

const {width: SCREEN_WIDTH} = Dimensions.get('window');
const SWIPE_THRESHOLD = 80;

type SingleDebateRouteProp = RouteProp<RootStackParamList, 'SingleDebate'>;

function formatScore(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

const SingleDebateScreen = () => {
  const route = useRoute<SingleDebateRouteProp>();
  const navigation = useNavigation();
  const insets = useSafeAreaInsets();
  const {match, debate} = route.params;
  const {token, isLoggedIn} = useAuth();

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

  // Card vote (swipe) state — totals for meter, and which cards user has voted this session
  const [cardVoteTotals, setCardVoteTotals] = useState<CardVoteTotals | null>(
    debate?.card_vote_totals ?? null,
  );
  /** Per-card vote counts for live Debate Pulse; updated when user votes (setCardVote response) */
  const [localCardVoteCounts, setLocalCardVoteCounts] = useState<
    Record<number, {upvotes: number; downvotes: number}>
  >({});
  const [votedCardIds, setVotedCardIds] = useState<Set<number>>(new Set());
  const [currentCardIndex, setCurrentCardIndex] = useState(0);
  const [swipeOffset, setSwipeOffset] = useState(0);
  const [swipeOverlay, setSwipeOverlay] = useState<'yes' | 'no' | null>(null);
  const [voteSubmitting, setVoteSubmitting] = useState(false);
  const swipeStartX = useRef(0);

  const cards: DebateCard[] = debate?.cards ?? [];
  const hasVotedAll =
    votedCardIds.size >= 3 || currentCardIndex >= cards.length;
  const showCardStack = cards.length > 0 && !hasVotedAll;

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

  // Keep card vote totals in sync with debate when refetched
  useEffect(() => {
    if (debate?.card_vote_totals) {
      setCardVoteTotals(debate.card_vote_totals);
    }
  }, [debate?.card_vote_totals]);

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

  const authGateContent = useMemo(() => {
    if (authGatePendingAction === 'swipe') {
      return {
        title: 'Login to vote',
        message: 'Sign in or create an account to vote on debate cards.',
      };
    }
    return {
      title: 'Join the conversation',
      message:
        'Sign in or create an account to reply, vote, or react on comments.',
    };
  }, [authGatePendingAction]);

  const submitCardVote = useCallback(
    async (cardId: number, voteType: 'upvote' | 'downvote') => {
      if (!debate?.id || !token) return;
      setVoteSubmitting(true);
      try {
        const counts = await setCardVote(token, debate.id, cardId, voteType);
        if (counts) {
          setCardVoteTotals({
            total_yes: counts.total_yes ?? 0,
            total_no: counts.total_no ?? 0,
          });
          // Update live Debate Pulse: this card’s counts from API
          setLocalCardVoteCounts(prev => ({
            ...prev,
            [cardId]: {
              upvotes: counts.yes_count,
              downvotes: counts.no_count,
            },
          }));
        }
        setVotedCardIds(prev => new Set(prev).add(cardId));
        setCurrentCardIndex(prev => Math.min(prev + 1, cards.length));
      } finally {
        setVoteSubmitting(false);
      }
    },
    [debate?.id, token, cards.length],
  );

  const handleSwipeVote = useCallback(
    (voteType: 'upvote' | 'downvote') => {
      if (!isLoggedIn) {
        setAuthGatePendingAction('swipe');
        return;
      }
      const card = cards[currentCardIndex];
      if (!card?.id || voteSubmitting) return;
      submitCardVote(card.id, voteType);
    },
    [isLoggedIn, cards, currentCardIndex, voteSubmitting, submitCardVote],
  );

  const panResponder = useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => showCardStack && !voteSubmitting,
        onMoveShouldSetPanResponder: (_, {dx}) => Math.abs(dx) > 10,
        onPanResponderGrant: () => {
          swipeStartX.current = 0;
        },
        onPanResponderMove: (_, {dx}) => {
          setSwipeOffset(dx);
          setSwipeOverlay(dx > 30 ? 'yes' : dx < -30 ? 'no' : null);
        },
        onPanResponderRelease: (_, {dx}) => {
          setSwipeOffset(0);
          setSwipeOverlay(null);
          if (dx > SWIPE_THRESHOLD) {
            handleSwipeVote('upvote');
          } else if (dx < -SWIPE_THRESHOLD) {
            handleSwipeVote('downvote');
          }
        },
      }),
    [showCardStack, voteSubmitting, handleSwipeVote],
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
                          subcomments: [
                            ...(sub.subcomments ?? []),
                            newComment,
                          ],
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
                  <ActivityIndicator size="small" color="#6b7280" />
                ) : (
                  <Ionicons
                    name="chevron-up"
                    size={20}
                    color={currentVote === 'upvote' ? '#22c55e' : '#6b7280'}
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
                  color={currentVote === 'downvote' ? '#ef4444' : '#6b7280'}
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
                <Ionicons name="arrow-undo-outline" size={14} color="#6b7280" />
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
                  <ActivityIndicator size="small" color="#6b7280" />
                ) : (
                  <Ionicons
                    name="add-circle-outline"
                    size={18}
                    color="#6b7280"
                  />
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
                  name={collapsedReplyIds.has(c.id) ? 'chevron-down' : 'chevron-up'}
                  size={16}
                  color="#6b7280"
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
          {isSub && c.subcomments && c.subcomments.length > 0 && (
            c.subcomments.map(sub => renderComment(sub, true))
          )}
        </View>
      </View>
    );
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={0}>
      {/* Top bar: back, FUCCI, icons */}
      <View style={[styles.topBar, {paddingTop: Math.max(insets.top, 8)}]}>
        <TouchableOpacity
          onPress={() => navigation.goBack()}
          style={styles.topBarButton}
          hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}>
          <Ionicons name="chevron-back" size={28} color="#1f2937" />
        </TouchableOpacity>
        <View style={styles.topBarRight}>
          <TouchableOpacity style={styles.iconButton}>
            <Ionicons name="person-outline" size={22} color="#1f2937" />
          </TouchableOpacity>
          <TouchableOpacity style={styles.iconButton}>
            <Ionicons name="notifications-outline" size={22} color="#1f2937" />
          </TouchableOpacity>
          <TouchableOpacity style={styles.iconButton}>
            <Ionicons name="ellipsis-horizontal" size={22} color="#1f2937" />
          </TouchableOpacity>
        </View>
      </View>

      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>
        {/* Header: team badges + score (hide score when pre-match) */}
        {match && (
          <View style={styles.matchHeader}>
            <View style={styles.teamBadge}>
              {match.teams?.home?.logo ? (
                <Image
                  source={{uri: match.teams.home.logo}}
                  style={styles.teamLogo}
                  resizeMode="contain"
                />
              ) : (
                <View style={styles.teamLogoPlaceholder} />
              )}
              <Text style={styles.teamName} numberOfLines={1}>
                {match.teams?.home?.name ?? 'Home'}
              </Text>
            </View>
            <View style={styles.scoreOrVs}>
              {match.goals?.home != null && match.goals?.away != null ? (
                <Text style={styles.scoreText}>
                  {match.goals.home} – {match.goals.away}
                </Text>
              ) : (
                <Text style={styles.headerVsText}>VS</Text>
              )}
            </View>
            <View style={styles.teamBadge}>
              {match.teams?.away?.logo ? (
                <Image
                  source={{uri: match.teams.away.logo}}
                  style={styles.teamLogo}
                  resizeMode="contain"
                />
              ) : (
                <View style={styles.teamLogoPlaceholder} />
              )}
              <Text style={styles.teamName} numberOfLines={1}>
                {match.teams?.away?.name ?? 'Away'}
              </Text>
            </View>
          </View>
        )}

        {/* Debate topic — above Debate Pulse */}
        <Text style={styles.headline}>{headline}</Text>

        {/* Live debate meter — Debate Pulse: summary bar + per-card breakdown */}
        {(() => {
          const hasCards = cards.length > 0;
          if (!hasCards) return null;
          // Use only yes (upvote) counts for Debate Pulse percentages; no votes excluded
          const cardYesVotes = cards.map(c => {
            const local = c.id != null ? localCardVoteCounts[c.id] : null;
            return local ? local.upvotes : (c.vote_counts?.upvotes ?? 0);
          });
          const totalYesVotes = cardYesVotes.reduce((a, b) => a + b, 0);
          const stanceColor = (stance: string) =>
            stance === 'agree'
              ? '#22c55e'
              : stance === 'disagree'
                ? '#ef4444'
                : '#eab308';
          return (
            <View style={styles.meterSection}>
              {/* Debate Pulse only — percentages per card, no separate summary bar */}
              {hasCards && (
                <View style={styles.debatePulseBox}>
                  <Text style={styles.debatePulseTitle}>Debate Pulse</Text>
                  {cards.map((card, i) => {
                    const yesVotes = cardYesVotes[i] ?? 0;
                    const pct =
                      totalYesVotes > 0
                        ? Math.round((yesVotes / totalYesVotes) * 100)
                        : 0;
                    const color = stanceColor(card.stance ?? 'wildcard');
                    return (
                      <View key={card.id ?? i} style={styles.debatePulseRow}>
                        <View style={styles.debatePulseLabelRow}>
                          <View
                            style={[
                              styles.debatePulseDot,
                              {backgroundColor: color},
                            ]}
                          />
                          <Text style={styles.debatePulseLabel}>
                            {card.title ||
                              (card.stance === 'agree'
                                ? 'Agree'
                                : card.stance === 'disagree'
                                  ? 'Disagree'
                                  : 'Wildcard')}
                          </Text>
                        </View>
                        <View style={styles.debatePulseBarRow}>
                          <View style={styles.debatePulseBarBg}>
                            <View
                              style={[
                                styles.debatePulseBarFill,
                                {
                                  backgroundColor: color,
                                  width: `${Math.min(100, pct)}%`,
                                },
                              ]}
                            />
                          </View>
                          <Text style={styles.debatePulsePct}>{pct}%</Text>
                        </View>
                      </View>
                    );
                  })}
                </View>
              )}
            </View>
          );
        })()}

        {/* Stacked cards — only show when user hasn't voted on all 3 */}
        {showCardStack && cards[currentCardIndex] && (
          <View style={styles.cardStackSection}>
            {/* Back cards (peek) — layered behind */}
            {cards
              .slice(currentCardIndex + 1, currentCardIndex + 3)
              .map((card, i) => (
                <View
                  key={card.id ?? i}
                  style={[
                    styles.stackedCard,
                    styles.stackedCardBack,
                    {top: 8 + i * 8, zIndex: 2 + i},
                  ]}>
                  <Text style={styles.stackedCardTitle}>{card.title}</Text>
                  <Text style={styles.stackedCardDesc} numberOfLines={2}>
                    {card.description}
                  </Text>
                </View>
              ))}
            {/* Top card — swipeable */}
            <View
              style={[
                styles.stackedCard,
                styles.stackedCardTop,
                {
                  transform: [{translateX: swipeOffset}],
                  zIndex: 10,
                },
              ]}
              {...panResponder.panHandlers}>
              {swipeOverlay && (
                <View
                  style={[
                    styles.swipeOverlay,
                    swipeOverlay === 'yes'
                      ? styles.swipeOverlayYes
                      : styles.swipeOverlayNo,
                  ]}>
                  <Ionicons
                    name={swipeOverlay === 'yes' ? 'thumbs-up' : 'thumbs-down'}
                    size={64}
                    color="#fff"
                  />
                </View>
              )}
              <Text style={styles.stackedCardTitle}>
                {cards[currentCardIndex].title}
              </Text>
              <Text style={styles.stackedCardDesc}>
                {cards[currentCardIndex].description}
              </Text>
              <Text style={styles.swipeHint}>Swipe right 👍 or left 👎</Text>
            </View>
          </View>
        )}

        {/* Comments — seeded viewpoints appear here as comments (no voting UI) */}
        <View style={styles.commentsHeader}>
          <TouchableOpacity style={styles.sortRow}>
            <Text style={styles.sortLabel}>Top Comments</Text>
            <Ionicons name="chevron-down" size={18} color="#6b7280" />
          </TouchableOpacity>
          <TouchableOpacity>
            <Ionicons name="filter" size={20} color="#6b7280" />
          </TouchableOpacity>
        </View>

        {/* Comment list — loading, error with Retry, or list (006 US1) */}
        {commentsLoading && (
          <View style={styles.commentsLoading}>
            <ActivityIndicator size="small" color="#6b7280" />
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

      {/* Fixed comment input */}
      {commentSubmitError && (
        <View style={styles.commentSubmitError}>
          <Text style={styles.commentSubmitErrorText}>
            {commentSubmitError}
          </Text>
          <TouchableOpacity onPress={() => setCommentSubmitError(null)}>
            <Text style={styles.commentSubmitErrorDismiss}>Dismiss</Text>
          </TouchableOpacity>
        </View>
      )}
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
                <Ionicons name="close" size={18} color="#6b7280" />
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
            placeholderTextColor="#9ca3af"
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
            <ActivityIndicator size="small" color="#007AFF" />
          ) : (
            <Ionicons name="send" size={20} color="#007AFF" />
          )}
        </TouchableOpacity>
      </View>

      <AuthGateModal
        visible={authGatePendingAction != null}
        onDismiss={() => setAuthGatePendingAction(null)}
        onLogin={() => {
          const pending = authGatePendingAction;
          setAuthGatePendingAction(null);
          rootNavigate('Login', {
            returnToDebate:
              match && debate
                ? {match, debate, pendingAction: pending ?? undefined}
                : undefined,
          });
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
    backgroundColor: '#ffffff',
  },
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 8,
    paddingTop: 8,
    paddingBottom: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#e5e7eb',
    backgroundColor: '#ffffff',
  },
  topBarButton: {
    padding: 4,
  },
  logoTitle: {
    fontSize: 20,
    fontWeight: '700',
    color: '#1f2937',
    letterSpacing: 0.5,
  },
  topBarRight: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  iconButton: {
    padding: 4,
  },
  scrollView: {
    flex: 1,
    backgroundColor: '#ffffff',
  },
  scrollContent: {
    paddingHorizontal: 20,
    paddingTop: 20,
    paddingBottom: 24,
  },
  matchHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 16,
    paddingVertical: 8,
  },
  teamBadge: {
    flex: 1,
    alignItems: 'center',
    maxWidth: 100,
  },
  teamLogo: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#f3f4f6',
  },
  teamLogoPlaceholder: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#e5e7eb',
  },
  teamName: {
    fontSize: 11,
    color: '#6b7280',
    marginTop: 4,
    textAlign: 'center',
  },
  scoreOrVs: {
    paddingHorizontal: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  scoreText: {
    fontSize: 18,
    fontWeight: '700',
    color: '#1f2937',
  },
  headerVsText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#9ca3af',
  },
  meterSection: {
    marginBottom: 20,
  },
  debatePulseBox: {
    marginTop: 16,
    backgroundColor: '#f3f4f6',
    borderRadius: 12,
    padding: 16,
  },
  debatePulseTitle: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1f2937',
    marginBottom: 12,
  },
  debatePulseRow: {
    marginBottom: 12,
  },
  debatePulseLabelRow: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    marginBottom: 6,
  },
  debatePulseDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    marginRight: 8,
    marginTop: 4,
  },
  debatePulseLabel: {
    flex: 1,
    fontSize: 14,
    color: '#374151',
    lineHeight: 20,
  },
  debatePulseBarRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginLeft: 18,
  },
  debatePulseBarBg: {
    flex: 1,
    height: 6,
    backgroundColor: '#e5e7eb',
    borderRadius: 3,
    overflow: 'hidden',
  },
  debatePulseBarFill: {
    height: '100%',
    borderRadius: 3,
  },
  debatePulsePct: {
    fontSize: 14,
    fontWeight: '600',
    color: '#374151',
    marginLeft: 8,
    minWidth: 36,
    textAlign: 'right',
  },
  meterBar: {
    flexDirection: 'row',
    height: 8,
    borderRadius: 4,
    overflow: 'hidden',
    backgroundColor: '#e5e7eb',
  },
  meterFillYes: {
    backgroundColor: '#22c55e',
    minWidth: 0,
  },
  meterFillNo: {
    backgroundColor: '#ef4444',
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
    color: '#6b7280',
  },
  meterLabelNo: {
    fontSize: 12,
    color: '#6b7280',
  },
  cardStackSection: {
    marginBottom: 24,
    minHeight: 160,
    position: 'relative',
  },
  stackedCard: {
    position: 'absolute',
    left: 20,
    right: 20,
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    borderWidth: 1,
    borderColor: '#e5e7eb',
  },
  stackedCardBack: {
    top: 8,
    marginLeft: 8,
    marginRight: 28,
    opacity: 0.85,
  },
  stackedCardTop: {
    top: 0,
  },
  swipeOverlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  swipeOverlayYes: {
    backgroundColor: 'rgba(34, 197, 94, 0.85)',
  },
  swipeOverlayNo: {
    backgroundColor: 'rgba(239, 68, 68, 0.85)',
  },
  swipeHint: {
    fontSize: 12,
    color: '#9ca3af',
    marginTop: 8,
    textAlign: 'center',
  },
  stackedCardTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1f2937',
    marginBottom: 6,
  },
  stackedCardDesc: {
    fontSize: 14,
    color: '#4b5563',
    lineHeight: 20,
  },
  headline: {
    fontSize: 20,
    fontWeight: '700',
    color: '#1f2937',
    lineHeight: 28,
    marginBottom: 24,
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
    backgroundColor: '#f3f4f6',
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
    color: '#374151',
    marginBottom: 8,
    maxWidth: 100,
    textAlign: 'center',
  },
  voteBarBg: {
    height: 6,
    width: '100%',
    maxWidth: 100,
    backgroundColor: '#e5e7eb',
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
    color: '#6b7280',
    marginTop: 4,
  },
  vsBadge: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: '#e5e7eb',
    alignItems: 'center',
    justifyContent: 'center',
    marginHorizontal: 8,
  },
  vsText: {
    fontSize: 12,
    fontWeight: '800',
    color: '#374151',
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
    borderTopColor: '#e5e7eb',
  },
  voteNowLinkLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#007AFF',
  },
  stanceCardsSection: {
    marginBottom: 28,
  },
  stanceCard: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
    borderColor: '#e5e7eb',
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
    color: '#fff',
    textTransform: 'uppercase',
  },
  stanceCardTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#1f2937',
    marginBottom: 8,
    lineHeight: 22,
  },
  stanceCardDescription: {
    fontSize: 14,
    color: '#4b5563',
    lineHeight: 20,
    marginBottom: 12,
  },
  stanceVoteBarBg: {
    height: 8,
    backgroundColor: '#e5e7eb',
    borderRadius: 4,
    overflow: 'hidden',
  },
  stanceVoteBarFill: {
    height: '100%',
    borderRadius: 4,
  },
  stanceVotePct: {
    fontSize: 12,
    color: '#6b7280',
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
    borderTopColor: '#e5e7eb',
  },
  stanceVoteNowLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#007AFF',
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
    fontSize: 16,
    fontWeight: '600',
    color: '#1f2937',
  },
  commentRow: {
    flexDirection: 'row',
    marginBottom: 20,
  },
  subcommentRow: {
    marginLeft: 24,
    marginBottom: 12,
  },
  commentAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: '#e5e7eb',
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
    color: '#6b7280',
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
    fontWeight: '600',
    color: '#1f2937',
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
    fontWeight: '600',
    color: '#059669',
    minWidth: 32,
    textAlign: 'center',
  },
  commentContent: {
    fontSize: 14,
    color: '#4b5563',
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
    color: '#6b7280',
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
    backgroundColor: '#f3f4f6',
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
    color: '#6b7280',
    fontWeight: '500',
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
    color: '#6b7280',
  },
  commentsError: {
    paddingVertical: 24,
    alignItems: 'center',
    gap: 12,
  },
  commentsErrorText: {
    fontSize: 14,
    color: '#6b7280',
    textAlign: 'center',
  },
  retryButton: {
    paddingHorizontal: 20,
    paddingVertical: 10,
    backgroundColor: '#e5e7eb',
    borderRadius: 8,
  },
  retryButtonText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#374151',
  },
  commentsEmpty: {
    fontSize: 14,
    color: '#6b7280',
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
    color: '#6b7280',
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
    backgroundColor: '#fef2f2',
    borderTopWidth: 1,
    borderTopColor: '#fecaca',
  },
  commentSubmitErrorText: {
    fontSize: 13,
    color: '#b91c1c',
    flex: 1,
  },
  commentSubmitErrorDismiss: {
    fontSize: 13,
    color: '#6b7280',
    marginLeft: 8,
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#ffffff',
    borderTopWidth: 1,
    borderTopColor: '#e5e7eb',
    gap: 10,
  },
  inputAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: '#e5e7eb',
    alignItems: 'center',
    justifyContent: 'center',
  },
  inputAvatarText: {
    fontSize: 14,
    fontWeight: '700',
    color: '#6b7280',
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
    color: '#6b7280',
  },
  input: {
    minHeight: 40,
    maxHeight: 100,
    backgroundColor: '#f3f4f6',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 15,
    color: '#1f2937',
  },
  inputCharCount: {
    fontSize: 11,
    color: '#9ca3af',
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
