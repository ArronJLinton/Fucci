import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  TouchableOpacity,
  TextInput,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RootStackParamList } from '../types/navigation';
import type { Match } from '../types/match';
import type {
  DebateResponse,
  DebateType,
  MockComment,
} from '../types/debate';
import {
  fetchDebatesByMatch,
  fetchDebateById,
  createDebate,
} from '../services/api';

const MOCK_COMMENTS: MockComment[] = [
  { id: '1', username: 'MrAficionado', content: 'He\'s more impactful and is a game changer for Madrid!', upvotes: 2700, replies: 12 },
  { id: '2', username: 'GoPSG', content: 'Haaland is a goal machine.', upvotes: 1100, replies: 5 },
  { id: '3', username: 'FutbolExpert', content: 'Both are world class—depends on the system.', upvotes: 827, replies: 0 },
];

const FINISHED_STATUSES = ['FT', 'AET', 'PEN', 'FT_PEN', 'AET_PEN', 'AWD', 'WO', 'CANC', 'ABD', 'PST'];

function getDefaultDebateType(match: Match): DebateType {
  const short = match?.fixture?.status?.short ?? '';
  return FINISHED_STATUSES.includes(short) ? 'post_match' : 'pre_match';
}

interface DebateScreenProps {
  match: Match;
  stackNavigation?: NativeStackNavigationProp<RootStackParamList>;
}

const DebateScreen: React.FC<DebateScreenProps> = ({ match, stackNavigation }) => {
  const fallbackNav = (stackNavigation ?? null) as NativeStackNavigationProp<RootStackParamList> | null;
  const stackNav = fallbackNav;

  const [debateData, setDebateData] = useState<DebateResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [debateType, setDebateType] = useState<DebateType>(() => getDefaultDebateType(match));
  const [commentInput, setCommentInput] = useState('');
  const [mockComments, setMockComments] = useState<MockComment[]>(MOCK_COMMENTS);
  const [mockVotePercentages, setMockVotePercentages] = useState<Record<string, number>>({
    agree: 55,
    disagree: 45,
    wildcard: 0,
  });

  const openSingleDebate = (selectedCardIndex: number) => {
    if (!debateData || !stackNav) return;
    stackNav.navigate('SingleDebate', { match, debate: debateData, selectedCardIndex });
  };

  const loadDebateForType = useCallback(
    async (type: DebateType) => {
      if (!match?.fixture?.id) {
        setError('Invalid match: missing fixture ID');
        setIsLoading(false);
        return;
      }
      setError(null);
      setIsLoading(true);
      setDebateData(null);
      try {
        const list = await fetchDebatesByMatch(match.fixture.id);
        const existing = list.find((d) => d.debate_type === type);
        if (existing) {
          const full = await fetchDebateById(existing.id);
          setDebateData(full ?? null);
        } else {
          setIsLoading(false);
          setIsGenerating(true);
          const created = await createDebate(match.fixture.id, type);
          setDebateData(created ?? null);
          if (!created) {
            setError('Could not generate debate. Try again.');
          }
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to load debate';
        setError(msg);
        setDebateData(null);
      } finally {
        setIsLoading(false);
        setIsGenerating(false);
      }
    },
    [match?.fixture?.id],
  );

  useEffect(() => {
    loadDebateForType(debateType);
  }, [debateType, loadDebateForType]);

  useEffect(() => {
    const defaultType = getDefaultDebateType(match);
    setDebateType((prev) => (prev !== defaultType ? defaultType : prev));
  }, [match?.fixture?.id, match?.fixture?.status?.short]);

  const getStanceColor = (
    stance: 'agree' | 'disagree' | 'wildcard' | DebateType,
  ) => {
    switch (stance) {
      case 'agree':
        return '#4CAF50';
      case 'disagree':
        return '#F44336';
      case 'wildcard':
        return '#FF9800';
      default:
        return '#666';
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

  const handleVotePress = (stance: string) => {
    if (!debateData?.cards?.length) return;
    const key = stance as keyof typeof mockVotePercentages;
    setMockVotePercentages((prev) => {
      const next = { ...prev };
      next[key] = (next[key] || 0) + 5;
      const total = Object.values(next).reduce((a, b) => a + b, 0);
      Object.keys(next).forEach((k) => {
        next[k] = Math.round((next[k] / total) * 100);
      });
      return next;
    });
  };

  const handleAddComment = () => {
    if (!commentInput.trim()) return;
    setMockComments((prev) => [
      ...prev,
      {
        id: String(Date.now()),
        username: 'You',
        content: commentInput.trim(),
        upvotes: 0,
        replies: 0,
      },
    ]);
    setCommentInput('');
  };

  const showLoading = isLoading || isGenerating;
  const loadingMessage = isGenerating
    ? 'Generating debate...'
    : 'Loading debate...';

  if (showLoading) {
    return (
      <View style={styles.centerContainer}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>{loadingMessage}</Text>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.errorText}>{error}</Text>
      </View>
    );
  }

  if (!debateData) {
    return (
      <View style={styles.centerContainer}>
        <Text style={styles.noDataText}>No debates yet</Text>
        <Text style={styles.emptySubtext}>
          Debates for this match haven't been generated yet.
        </Text>
      </View>
    );
  }

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={80}
    >
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.header}>
          <Text style={styles.headerTitle}>Match Debate</Text>
          <Text style={styles.headerSubtitle}>
            {match.teams.home.name} vs {match.teams.away.name}
          </Text>
          <Text style={styles.debatePhaseLabel}>
            {debateType === 'pre_match' ? 'Pre-Match' : 'Post-Match'}
          </Text>
        </View>

        <View style={styles.content}>
          <TouchableOpacity
            style={styles.promptContainer}
            activeOpacity={0.9}
            onPress={() => openSingleDebate(0)}
          >
            <Text style={styles.promptHeadline}>{debateData.headline}</Text>
            <Text style={styles.promptDescription}>
              {debateData.description}
            </Text>
            <View style={styles.joinConversationRow}>
              <Text style={styles.joinConversationLabel}>Join the conversation</Text>
              <Ionicons name="chevron-forward" size={18} color="#007AFF" />
            </View>
          </TouchableOpacity>

          {debateData.cards && debateData.cards.length > 0 ? (
            debateData.cards.map((card, index) => {
              const pct = mockVotePercentages[card.stance] ?? 0;
              return (
                <View key={card.stance} style={styles.debateCard}>
                  <TouchableOpacity
                    activeOpacity={0.85}
                    onPress={() => openSingleDebate(index)}
                  >
                    <View style={styles.cardHeader}>
                      <Text style={styles.stanceIcon}>
                        {getStanceIcon(card.stance)}
                      </Text>
                      <View
                        style={[
                          styles.stanceBadge,
                          { backgroundColor: getStanceColor(card.stance) },
                        ]}
                      >
                        <Text style={styles.stanceText}>
                          {card.stance.charAt(0).toUpperCase() +
                            card.stance.slice(1)}
                        </Text>
                      </View>
                    </View>
                    <Text style={styles.cardTitle}>{card.title}</Text>
                    <Text style={styles.cardDescription}>{card.description}</Text>
                    <View style={styles.voteBarContainer}>
                      <View style={styles.voteBarBg}>
                        <View
                          style={[
                            styles.voteBarFill,
                            {
                              width: `${pct}%`,
                              backgroundColor: getStanceColor(card.stance),
                            },
                          ]}
                        />
                      </View>
                      <Text style={styles.votePct}>{pct}%</Text>
                    </View>
                  </TouchableOpacity>
                  <TouchableOpacity
                    style={styles.voteNowLinkRow}
                    activeOpacity={0.7}
                    onPress={() => handleVotePress(card.stance)}
                  >
                    <Text style={styles.voteNowLinkLabel}>Vote Now</Text>
                    <Ionicons name="chevron-forward" size={18} color="#007AFF" />
                  </TouchableOpacity>
                </View>
              );
            })
          ) : (
            <Text style={styles.noDebateText}>No debate cards available.</Text>
          )}

          {/* Comments section (mock data - T031) */}
          <View style={styles.commentsSection}>
            <Text style={styles.commentsSectionTitle}>Top Comments</Text>
            {mockComments.map((c) => (
              <View key={c.id} style={styles.commentCard}>
                <Text style={styles.commentUsername}>{c.username}</Text>
                <Text style={styles.commentContent}>{c.content}</Text>
                <View style={styles.commentMeta}>
                  <Text style={styles.commentUpvotes}>↑ {c.upvotes}</Text>
                  {c.replies != null && (
                    <Text style={styles.commentReplies}>
                      {c.replies} replies
                    </Text>
                  )}
                </View>
              </View>
            ))}
            <View style={styles.commentInputRow}>
              <TextInput
                style={styles.commentInput}
                placeholder="Write your opinion..."
                placeholderTextColor="#999"
                value={commentInput}
                onChangeText={setCommentInput}
                multiline
              />
              <TouchableOpacity
                style={styles.addCommentButton}
                onPress={handleAddComment}
              >
                <Text style={styles.addCommentButtonText}>Add</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f5f5f5',
  },
  scrollView: {
    flex: 1,
  },
  scrollContent: {
    paddingBottom: 24,
  },
  centerContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f5f5f5',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 16,
    color: '#666',
  },
  errorText: {
    fontSize: 16,
    color: '#ff3b30',
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  noDataText: {
    fontSize: 18,
    color: '#333',
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  emptySubtext: {
    fontSize: 14,
    color: '#666',
    marginTop: 8,
    paddingHorizontal: 20,
    textAlign: 'center',
  },
  header: {
    backgroundColor: '#fff',
    paddingHorizontal: 16,
    paddingVertical: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#e0e0e0',
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#333',
    marginBottom: 4,
  },
  headerSubtitle: {
    fontSize: 16,
    color: '#666',
    marginBottom: 8,
  },
  debatePhaseLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#007AFF',
  },
  content: {
    padding: 16,
  },
  promptContainer: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 3.84,
    elevation: 5,
  },
  promptHeadline: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#333',
    marginBottom: 8,
    lineHeight: 24,
  },
  promptDescription: {
    fontSize: 14,
    color: '#666',
    lineHeight: 20,
  },
  joinConversationRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    marginTop: 14,
    paddingTop: 14,
    borderTopWidth: 1,
    borderTopColor: '#e8e8e8',
  },
  joinConversationLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#007AFF',
  },
  debateCard: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 3.84,
    elevation: 5,
  },
  cardHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  stanceIcon: {
    fontSize: 24,
  },
  stanceBadge: {
    paddingHorizontal: 12,
    paddingVertical: 4,
    borderRadius: 12,
  },
  stanceText: {
    fontSize: 12,
    fontWeight: '600',
    color: '#fff',
    textTransform: 'uppercase',
  },
  cardTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
    marginBottom: 8,
    lineHeight: 22,
  },
  cardDescription: {
    fontSize: 14,
    color: '#666',
    lineHeight: 20,
    marginBottom: 12,
  },
  voteBarContainer: {
    marginBottom: 12,
  },
  voteBarBg: {
    height: 8,
    backgroundColor: '#e0e0e0',
    borderRadius: 4,
    overflow: 'hidden',
  },
  voteBarFill: {
    height: '100%',
    borderRadius: 4,
  },
  votePct: {
    fontSize: 12,
    color: '#666',
    marginTop: 4,
  },
  voteNowLinkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    marginTop: 12,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: '#e8e8e8',
  },
  voteNowLinkLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#007AFF',
  },
  noDebateText: {
    fontSize: 16,
    color: '#666',
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  commentsSection: {
    marginTop: 24,
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 3.84,
    elevation: 5,
  },
  commentsSectionTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333',
    marginBottom: 12,
  },
  commentCard: {
    borderBottomWidth: 1,
    borderBottomColor: '#f0f0f0',
    paddingVertical: 12,
  },
  commentUsername: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333',
    marginBottom: 4,
  },
  commentContent: {
    fontSize: 14,
    color: '#666',
    lineHeight: 20,
  },
  commentMeta: {
    flexDirection: 'row',
    marginTop: 6,
    gap: 12,
  },
  commentUpvotes: {
    fontSize: 12,
    color: '#4CAF50',
  },
  commentReplies: {
    fontSize: 12,
    color: '#999',
  },
  commentInputRow: {
    flexDirection: 'row',
    alignItems: 'flex-end',
    marginTop: 16,
    gap: 8,
  },
  commentInput: {
    flex: 1,
    borderWidth: 1,
    borderColor: '#e0e0e0',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
    color: '#333',
    minHeight: 44,
    maxHeight: 100,
  },
  addCommentButton: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderRadius: 8,
    justifyContent: 'center',
  },
  addCommentButtonText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: '600',
  },
});

export default DebateScreen;
