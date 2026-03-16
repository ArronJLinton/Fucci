import React, {useState, useEffect, useCallback} from 'react';
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
} from 'react-native';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import type {RootStackParamList} from '../types/navigation';
import type {DebateComment} from '../types/debate';
import {listComments} from '../services/api';

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

  const [comments, setComments] = useState<DebateComment[]>([]);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const [commentsError, setCommentsError] = useState<string | null>(null);
  const [commentInput, setCommentInput] = useState('');

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

  const headline = debate?.headline ?? 'Debate';

  const handleAddComment = () => {
    if (!commentInput.trim()) return;
    // TODO Phase 4: POST comment via API
    setCommentInput('');
  };

  const renderComment = (c: DebateComment, isSub?: boolean) => (
    <View key={c.id} style={[styles.commentRow, isSub && styles.subcommentRow]}>
      <View style={styles.commentAvatar}>
        {c.user_avatar_url ? (
          <Image source={{uri: c.user_avatar_url}} style={styles.commentAvatarImage} />
        ) : (
          <Text style={styles.commentAvatarText}>
            {(c.user_display_name || '?').charAt(0).toUpperCase()}
          </Text>
        )}
      </View>
      <View style={styles.commentBody}>
        <View style={styles.commentMetaRow}>
          <Text style={styles.commentUsername}>{c.user_display_name || 'User'}</Text>
          <Text style={styles.commentUpvotes}>
            {c.net_score >= 0 ? '+' : ''}{formatScore(c.net_score)}
          </Text>
        </View>
        <Text style={styles.commentContent}>{c.content}</Text>
        {c.reactions && c.reactions.length > 0 && (
          <View style={styles.reactionsRow}>
            {c.reactions.map((r, i) => (
              <Text key={`${c.id}-${i}`} style={styles.reactionChip}>
                {r.emoji} {r.count}
              </Text>
            ))}
          </View>
        )}
        {c.subcomments && c.subcomments.length > 0 && c.subcomments.map(sub => renderComment(sub, true))}
      </View>
    </View>
  );

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
        {/* Debate question */}
        <Text style={styles.headline}>{headline}</Text>

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
        {!commentsLoading && !commentsError && comments.length === 0 && debate?.id != null && (
          <Text style={styles.commentsEmpty}>No comments yet.</Text>
        )}
        {!commentsLoading && !commentsError && comments.map(c => renderComment(c))}

        <View style={styles.bottomSpacer} />
      </ScrollView>

      {/* Fixed comment input */}
      <View
        style={[
          styles.inputRow,
          {paddingBottom: Math.max(insets.bottom, 12) + 12},
        ]}>
        <View style={styles.inputAvatar}>
          <Text style={styles.inputAvatarText}>Y</Text>
        </View>
        <TextInput
          style={styles.input}
          placeholder="Write a comment..."
          placeholderTextColor="#9ca3af"
          value={commentInput}
          onChangeText={setCommentInput}
          multiline
          maxLength={500}
        />
        <TouchableOpacity onPress={handleAddComment} style={styles.sendButton}>
          <Ionicons name="send" size={20} color="#007AFF" />
        </TouchableOpacity>
        <TouchableOpacity style={styles.emojiButton}>
          <Ionicons name="happy-outline" size={22} color="#6b7280" />
        </TouchableOpacity>
      </View>
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
  commentUpvotes: {
    fontSize: 13,
    fontWeight: '600',
    color: '#059669',
  },
  commentContent: {
    fontSize: 14,
    color: '#4b5563',
    lineHeight: 20,
    marginBottom: 8,
  },
  reactionsRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
    marginTop: 6,
  },
  reactionChip: {
    fontSize: 13,
    color: '#6b7280',
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
  input: {
    flex: 1,
    minHeight: 40,
    maxHeight: 100,
    backgroundColor: '#f3f4f6',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    fontSize: 15,
    color: '#1f2937',
  },
  sendButton: {
    padding: 8,
  },
  emojiButton: {
    padding: 8,
  },
});

export default SingleDebateScreen;
