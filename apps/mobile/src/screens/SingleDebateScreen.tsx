import React, {useState} from 'react';
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
} from 'react-native';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import type {RootStackParamList} from '../types/navigation';
import type {DebateCard, MockComment} from '../types/debate';

type SingleDebateRouteProp = RouteProp<RootStackParamList, 'SingleDebate'>;

const MOCK_COMMENTS: (MockComment & {views?: number})[] = [
  {
    id: '1',
    username: 'MrAficionado',
    content: "He's more impactful and is a game changer for Madrid!",
    upvotes: 2700,
    replies: 12,
    views: 365,
  },
  {
    id: '2',
    username: 'GoPSG',
    content: 'Haaland is a goal machine.',
    upvotes: 1100,
    replies: 5,
    views: 212,
  },
  {
    id: '3',
    username: 'FutbolExpert',
    content: 'Both are world class—depends on the system.',
    upvotes: 827,
    replies: 0,
    views: 180,
  },
];

function formatUpvotes(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

const SingleDebateScreen = () => {
  const route = useRoute<SingleDebateRouteProp>();
  const navigation = useNavigation();
  const insets = useSafeAreaInsets();
  const {match, debate} = route.params;

  const [mockComments, setMockComments] = useState(MOCK_COMMENTS);
  const [commentInput, setCommentInput] = useState('');
  const [sideAPct, setSideAPct] = useState(61);
  const [sideBPct, setSideBPct] = useState(39);
  const [cardVotePcts, setCardVotePcts] = useState<Record<string, number>>({
    agree: 55,
    disagree: 45,
    wildcard: 0,
  });

  const headline = debate?.headline ?? 'Debate';
  const sideALabel = match?.teams?.home?.name ?? 'Yes';
  const sideBLabel = match?.teams?.away?.name ?? 'No';
  const cards: DebateCard[] = debate?.cards ?? [];

  const getStanceColor = (stance: string) => {
    switch (stance) {
      case 'agree':
        return '#4CAF50';
      case 'disagree':
        return '#F44336';
      case 'wildcard':
        return '#FF9800';
      default:
        return '#6b7280';
    }
  };

  const getStanceIcon = (stance: string) => {
    switch (stance) {
      case 'agree':
        return '👍';
      case 'disagree':
        return '👎';
      case 'wildcard':
        return '🎯';
      default:
        return '❓';
    }
  };

  const handleCardVote = (stance: string) => {
    const key = stance as keyof typeof cardVotePcts;
    setCardVotePcts(prev => {
      const next = {...prev};
      next[key] = (next[key] ?? 0) + 5;
      const total = Object.values(next).reduce((a, b) => a + b, 0);
      Object.keys(next).forEach(k => {
        next[k] = Math.round((next[k] / total) * 100);
      });
      return next;
    });
  };

  const handleVoteNow = () => {
    setSideAPct(p => Math.min(100, p + 2));
    setSideBPct(p => Math.max(0, p - 2));
  };

  const handleAddComment = () => {
    if (!commentInput.trim()) return;
    setMockComments(prev => [
      ...prev,
      {
        id: String(Date.now()),
        username: 'You',
        content: commentInput.trim(),
        upvotes: 0,
        replies: 0,
        views: 0,
      },
    ]);
    setCommentInput('');
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
        {/* Debate question */}
        <Text style={styles.headline}>{headline}</Text>

        {/* Two sides with VS */}
        <View style={styles.vsRow}>
          <View style={styles.playerBlock}>
            <View style={styles.avatarCircle}>
              {match?.teams?.home?.logo ? (
                <Image
                  source={{uri: match.teams.home.logo}}
                  style={styles.avatarImage}
                  resizeMode="contain"
                />
              ) : (
                <Ionicons name="shirt-outline" size={40} color="#6b7280" />
              )}
            </View>
            <Text style={styles.playerName} numberOfLines={1}>
              {sideALabel}
            </Text>
            <View style={styles.voteBarBg}>
              <View
                style={[
                  styles.voteBarFill,
                  {width: `${sideAPct}%`, backgroundColor: '#3B82F6'},
                ]}
              />
            </View>
            <Text style={styles.votePct}>{sideAPct}%</Text>
          </View>

          <View style={styles.vsBadge}>
            <Text style={styles.vsText}>VS</Text>
          </View>

          <View style={styles.playerBlock}>
            <View style={styles.avatarCircle}>
              {match?.teams?.away?.logo ? (
                <Image
                  source={{uri: match.teams.away.logo}}
                  style={styles.avatarImage}
                  resizeMode="contain"
                />
              ) : (
                <Ionicons name="shirt-outline" size={40} color="#6b7280" />
              )}
            </View>
            <Text style={styles.playerName} numberOfLines={1}>
              {sideBLabel}
            </Text>
            <View style={styles.voteBarBg}>
              <View
                style={[
                  styles.voteBarFill,
                  {width: `${sideBPct}%`, backgroundColor: '#3B82F6'},
                ]}
              />
            </View>
            <Text style={styles.votePct}>{sideBPct}%</Text>
          </View>
        </View>

        <TouchableOpacity
          style={styles.voteNowLinkRow}
          activeOpacity={0.7}
          onPress={handleVoteNow}
        >
          <Text style={styles.voteNowLinkLabel}>Vote Now</Text>
          <Ionicons name="chevron-forward" size={18} color="#007AFF" />
        </TouchableOpacity>

        {/* Agree / Disagree / Wildcard cards */}
        {cards.length > 0 && (
          <View style={styles.stanceCardsSection}>
            {cards.map(card => {
              const pct = cardVotePcts[card.stance] ?? 0;
              return (
                <View key={card.stance} style={styles.stanceCard}>
                  <View style={styles.stanceCardHeader}>
                    <Text style={styles.stanceCardIcon}>
                      {getStanceIcon(card.stance)}
                    </Text>
                    <View
                      style={[
                        styles.stanceBadge,
                        {backgroundColor: getStanceColor(card.stance)},
                      ]}>
                      <Text style={styles.stanceBadgeText}>
                        {card.stance.charAt(0).toUpperCase() +
                          card.stance.slice(1)}
                      </Text>
                    </View>
                  </View>
                  <Text style={styles.stanceCardTitle}>{card.title}</Text>
                  <Text style={styles.stanceCardDescription}>
                    {card.description}
                  </Text>
                  <View style={styles.stanceVoteBarBg}>
                    <View
                      style={[
                        styles.stanceVoteBarFill,
                        {
                          width: `${pct}%`,
                          backgroundColor: getStanceColor(card.stance),
                        },
                      ]}
                    />
                  </View>
                  <Text style={styles.stanceVotePct}>{pct}%</Text>
                  <TouchableOpacity
                    style={styles.stanceVoteNowRow}
                    activeOpacity={0.7}
                    onPress={() => handleCardVote(card.stance)}>
                    <Text style={styles.stanceVoteNowLabel}>Vote Now</Text>
                    <Ionicons
                      name="chevron-forward"
                      size={18}
                      color="#007AFF"
                    />
                  </TouchableOpacity>
                </View>
              );
            })}
          </View>
        )}

        {/* Top Comments header */}
        <View style={styles.commentsHeader}>
          <TouchableOpacity style={styles.sortRow}>
            <Text style={styles.sortLabel}>Top Comments</Text>
            <Ionicons name="chevron-down" size={18} color="#6b7280" />
          </TouchableOpacity>
          <TouchableOpacity>
            <Ionicons name="filter" size={20} color="#6b7280" />
          </TouchableOpacity>
        </View>

        {/* Comment list */}
        {mockComments.map(c => (
          <View key={c.id} style={styles.commentRow}>
            <View style={styles.commentAvatar}>
              <Text style={styles.commentAvatarText}>
                {(c.username || '?').charAt(0)}
              </Text>
            </View>
            <View style={styles.commentBody}>
              <View style={styles.commentMetaRow}>
                <Text style={styles.commentUsername}>{c.username}</Text>
                <Text style={styles.commentUpvotes}>
                  + {formatUpvotes(c.upvotes)}
                </Text>
              </View>
              <Text style={styles.commentContent}>{c.content}</Text>
              <View style={styles.commentActions}>
                <View style={styles.commentActionItem}>
                  <Ionicons name="eye-outline" size={14} color="#6b7280" />
                  <Text style={styles.commentActionText}>{c.views ?? 0}</Text>
                </View>
                <View style={styles.commentActionItem}>
                  <Ionicons
                    name="chatbubble-outline"
                    size={14}
                    color="#6b7280"
                  />
                  <Text style={styles.commentActionText}>{c.replies ?? 0}</Text>
                </View>
                <TouchableOpacity style={styles.commentActionItem}>
                  <Ionicons name="chevron-up" size={16} color="#6b7280" />
                </TouchableOpacity>
                <TouchableOpacity style={styles.commentActionItem}>
                  <Ionicons name="chevron-down" size={16} color="#6b7280" />
                </TouchableOpacity>
                <TouchableOpacity style={styles.commentActionItem}>
                  <Ionicons name="chevron-down" size={14} color="#6b7280" />
                </TouchableOpacity>
              </View>
            </View>
          </View>
        ))}

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
  commentAvatar: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: '#e5e7eb',
    alignItems: 'center',
    justifyContent: 'center',
    marginRight: 12,
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
