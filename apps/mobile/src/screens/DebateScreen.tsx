import React, { useState, useEffect, useCallback } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  ActivityIndicator,
  TouchableOpacity,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { Ionicons } from '@expo/vector-icons';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import type { RootStackParamList } from '../types/navigation';
import type { Match } from '../types/match';
import type { DebateResponse, DebateType } from '../types/debate';
import {
  fetchDebatesByMatch,
  fetchDebateById,
  createDebate,
  generateDebateSet,
} from '../services/api';

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

  const [debateList, setDebateList] = useState<DebateResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [debateType, setDebateType] = useState<DebateType>(() => getDefaultDebateType(match));

  const openSingleDebate = (debate: DebateResponse, selectedCardIndex: number = 0) => {
    if (!stackNav) return;
    stackNav.navigate('SingleDebate', { match, debate, selectedCardIndex });
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
      setDebateList([]);
      const matchId = match.fixture.id;
      const POLL_INTERVAL_MS = 3000;
      const POLL_TIMEOUT_MS = 60000;

      try {
        let list = await fetchDebatesByMatch(matchId, type);
        if (list.length > 0) {
          const fullDebates: DebateResponse[] = [];
          for (const item of list) {
            const full = await fetchDebateById(item.id);
            if (full) fullDebates.push(full);
          }
          setDebateList(fullDebates);
          return;
        }

        setIsGenerating(true);
        const setResult = await generateDebateSet(matchId, type, 3);
        if (setResult?.debates?.length) {
          setDebateList(setResult.debates);
          return;
        }
        if (setResult?.pending) {
          const deadline = Date.now() + POLL_TIMEOUT_MS;
          while (Date.now() < deadline) {
            await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
            list = await fetchDebatesByMatch(matchId, type);
            if (list.length > 0) {
              const fullDebates: DebateResponse[] = [];
              for (const item of list) {
                const full = await fetchDebateById(item.id);
                if (full) fullDebates.push(full);
              }
              setDebateList(fullDebates);
              return;
            }
          }
        }

        const created = await createDebate(matchId, type);
        if (created) {
          setDebateList([created]);
        } else {
          setError('Could not generate debate. Try again.');
        }
      } catch (err) {
        const msg = err instanceof Error ? err.message : 'Failed to load debate';
        setError(msg);
        setDebateList([]);
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

  if (!debateList.length) {
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
          {debateList.map((debate, index) => (
            <TouchableOpacity
              key={debate.id ?? index}
              style={styles.promptContainer}
              activeOpacity={0.9}
              onPress={() => openSingleDebate(debate, 0)}
            >
              <Text style={styles.promptHeadline}>{debate.headline}</Text>
              <Text style={styles.promptDescription}>
                {debate.description}
              </Text>
              <View style={styles.joinConversationRow}>
                <Text style={styles.joinConversationLabel}>Join the conversation</Text>
                <Ionicons name="chevron-forward" size={18} color="#007AFF" />
              </View>
            </TouchableOpacity>
          ))}
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
});

export default DebateScreen;
