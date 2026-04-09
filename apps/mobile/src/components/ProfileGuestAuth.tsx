import React, {useState} from 'react';
import {
  View,
  Text,
  TextInput,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Alert,
  Dimensions,
} from 'react-native';
import {LinearGradient} from 'expo-linear-gradient';
import {Ionicons} from '@expo/vector-icons';
import {useAuth} from '../context/AuthContext';
import {login} from '../services/api';
import {
  launchGoogleAuthBrowserFlow,
  resolvePostGoogleAuthRoute,
} from '../services/googleAuth';
import {rootNavigate} from '../navigation/rootNavigation';
import {dispatchAfterSignInSuccess} from '../navigation/authNavigationActions';
import type {ReturnToDebateParams} from '../types/navigation';

export type ProfileGuestAuthProps = {
  /** When set (e.g. from debate auth gate), resume this screen after sign-in. */
  returnToDebate?: ReturnToDebateParams;
};

/**
 * Logged-out Profile tab: full auth landing aligned with product login visual language
 * (dark pitch, FUCCI wordmark, social + email flow).
 */
export default function ProfileGuestAuth({
  returnToDebate,
}: ProfileGuestAuthProps) {
  const {setAuth} = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleEmailLogin = async () => {
    if (!email.trim() || !password) {
      setError('Enter your email and password.');
      return;
    }
    setError(null);
    setBusy(true);
    try {
      const result = await login({email: email.trim(), password});
      if (result.ok) {
        await setAuth(result.data.token, result.data.user);
        dispatchAfterSignInSuccess({returnToDebate});
        return;
      }
      setError(result.status === 401 ? 'Invalid credentials.' : result.message);
    } catch {
      setError('Something went wrong. Try again.');
    } finally {
      setBusy(false);
    }
  };

  const handleGoogle = async () => {
    setError(null);
    setBusy(true);
    try {
      const authResult = await launchGoogleAuthBrowserFlow();
      if (authResult.kind === 'cancel') {
        return;
      }
      if (authResult.kind === 'error') {
        setError(authResult.message);
        return;
      }
      await setAuth(authResult.token, authResult.user);
      const destination = resolvePostGoogleAuthRoute(authResult.isNew);
      if (destination === 'CreatePlayerProfile') {
        dispatchAfterSignInSuccess({replaceWithCreatePlayerProfile: true});
      } else {
        dispatchAfterSignInSuccess({returnToDebate});
      }
    } catch (e) {
      setError(
        e instanceof Error ? e.message : 'Google sign-in failed. Try again.',
      );
    } finally {
      setBusy(false);
    }
  };

  const handleApple = () => {
    Alert.alert(
      'Coming soon',
      'Apple Sign-In is not available yet. Use Google or email to continue.',
    );
  };

  return (
    <KeyboardAvoidingView
      style={styles.flex}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={Platform.OS === 'ios' ? 24 : 0}>
      <ScrollView
        style={styles.flex}
        contentContainerStyle={[
          styles.scroll,
          {minHeight: Dimensions.get('window').height - 100},
        ]}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>
        <View style={styles.bgWrap}>
          <LinearGradient
            colors={['#020617', '#0f172a', '#022c22', '#030712']}
            start={{x: 0, y: 0}}
            end={{x: 1, y: 1}}
            style={StyleSheet.absoluteFill}
          />
          <PitchOverlay />
        </View>

        <View style={styles.inner}>
          <View style={styles.brandRow}>
            <Ionicons name="football" size={36} color="#c7f349" />
            <Text style={styles.brandWordmark}>FUCCI</Text>
          </View>
          <Text style={styles.tagline}>
            Please sign in or register to access profile features.
          </Text>

          {error ? <Text style={styles.errorText}>{error}</Text> : null}

          <TouchableOpacity
            style={[styles.socialBtn, busy && styles.disabled]}
            onPress={handleGoogle}
            disabled={busy}
            accessibilityRole="button"
            accessibilityLabel="Continue with Google">
            <View style={styles.socialInner}>
              <Text style={styles.googleG}>G</Text>
              <Text style={styles.socialLabel}>Continue with Google</Text>
            </View>
            {busy ? (
              <ActivityIndicator color="#e2e8f0" style={styles.socialSpinner} />
            ) : null}
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.socialBtn, busy && styles.disabled]}
            onPress={handleApple}
            disabled={busy}
            accessibilityRole="button"
            accessibilityLabel="Continue with Apple">
            <View style={styles.socialInner}>
              <Ionicons name="logo-apple" size={22} color="#f8fafc" />
              <Text style={styles.socialLabel}>Continue with Apple</Text>
            </View>
          </TouchableOpacity>

          <View style={styles.orRow}>
            <View style={styles.orLine} />
            <Text style={styles.orText}>OR</Text>
            <View style={styles.orLine} />
          </View>

          <Text style={styles.fieldLabel}>Email address</Text>
          <TextInput
            style={styles.input}
            placeholder="manager@fucci.fc"
            placeholderTextColor="#64748b"
            value={email}
            onChangeText={setEmail}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            editable={!busy}
          />

          <View style={styles.passwordLabelRow}>
            <Text style={styles.fieldLabel}>Password</Text>
            <TouchableOpacity
              onPress={() => rootNavigate('ForgotPassword')}
              disabled={busy}
              hitSlop={{top: 8, bottom: 8, left: 8, right: 8}}>
              <Text style={styles.forgotLink}>Forgot tactical plan?</Text>
            </TouchableOpacity>
          </View>
          <TextInput
            style={styles.input}
            placeholder="••••••••"
            placeholderTextColor="#64748b"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            editable={!busy}
          />
          <Text style={styles.passwordHint}>
            Min 8 characters, one letter, one number.
          </Text>

          <TouchableOpacity
            activeOpacity={0.92}
            onPress={handleEmailLogin}
            disabled={busy}
            style={styles.arenaBtnOuter}>
            <LinearGradient
              colors={['#ecfccb', '#bef264', '#a3e635']}
              start={{x: 0, y: 0.5}}
              end={{x: 1, y: 0.5}}
              style={styles.arenaGradient}>
              {busy ? (
                <ActivityIndicator color="#14532d" />
              ) : (
                <Text style={styles.arenaText}>Login to arena</Text>
              )}
            </LinearGradient>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.footerRow}
            onPress={() => rootNavigate('SignUp')}
            disabled={busy}>
            <Text style={styles.footerMuted}>{"Don't have an account? "}</Text>
            <Text style={styles.footerAccent}>Sign up</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

function PitchOverlay() {
  return (
    <View style={styles.pitchOverlay} pointerEvents="none">
      <View style={[styles.pitchLine, styles.pitchLineH, {top: '22%'}]} />
      <View style={[styles.pitchLine, styles.pitchLineH, {top: '50%'}]} />
      <View style={[styles.pitchLine, styles.pitchLineH, {top: '78%'}]} />
      <View style={[styles.pitchLine, styles.pitchLineV, {left: '50%'}]} />
      <View style={styles.centerCircle} />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: {
    flex: 1,
  },
  scroll: {
    flexGrow: 1,
    paddingBottom: 32,
  },
  bgWrap: {
    ...StyleSheet.absoluteFillObject,
    overflow: 'hidden',
  },
  pitchOverlay: {
    ...StyleSheet.absoluteFillObject,
    opacity: 0.12,
  },
  pitchLine: {
    position: 'absolute',
    backgroundColor: '#c7f349',
  },
  pitchLineH: {
    height: 1,
    left: '6%',
    right: '6%',
  },
  pitchLineV: {
    width: 1,
    top: '12%',
    bottom: '12%',
    marginLeft: -0.5,
  },
  centerCircle: {
    position: 'absolute',
    width: 120,
    height: 120,
    borderRadius: 60,
    borderWidth: 1,
    borderColor: '#c7f349',
    left: '50%',
    top: '42%',
    marginLeft: -60,
    marginTop: -60,
    opacity: 0.35,
  },
  inner: {
    paddingHorizontal: 22,
    paddingTop: 8,
    maxWidth: 440,
    alignSelf: 'center',
    width: '100%',
  },
  brandRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 10,
    marginBottom: 10,
  },
  brandWordmark: {
    fontSize: 36,
    fontWeight: '900',
    fontStyle: 'italic',
    letterSpacing: 2,
    color: '#d9f99d',
  },
  tagline: {
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 1.2,
    color: '#e2e8f0',
    textTransform: 'uppercase',
    marginBottom: 22,
    lineHeight: 16,
  },
  errorText: {
    color: '#fca5a5',
    fontSize: 14,
    marginBottom: 12,
  },
  socialBtn: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    paddingVertical: 14,
    paddingHorizontal: 16,
    marginBottom: 12,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  disabled: {
    opacity: 0.65,
  },
  socialInner: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
  },
  googleG: {
    fontSize: 18,
    fontWeight: '800',
    color: '#f8fafc',
    width: 24,
    textAlign: 'center',
  },
  socialLabel: {
    fontSize: 16,
    fontWeight: '600',
    color: '#f1f5f9',
  },
  socialSpinner: {
    marginRight: 4,
  },
  orRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginVertical: 18,
    gap: 12,
  },
  orLine: {
    flex: 1,
    height: 1,
    backgroundColor: '#334155',
  },
  orText: {
    fontSize: 11,
    fontWeight: '800',
    color: '#94a3b8',
    letterSpacing: 2,
  },
  fieldLabel: {
    fontSize: 10,
    fontWeight: '800',
    color: '#94a3b8',
    letterSpacing: 1.4,
    textTransform: 'uppercase',
    marginBottom: 8,
  },
  passwordLabelRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginTop: 4,
  },
  forgotLink: {
    fontSize: 10,
    fontWeight: '800',
    color: '#67e8f9',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
  },
  input: {
    backgroundColor: '#020617',
    borderWidth: 1,
    borderColor: '#1e293b',
    borderRadius: 10,
    paddingHorizontal: 14,
    paddingVertical: 14,
    fontSize: 16,
    color: '#f8fafc',
    marginBottom: 4,
  },
  passwordHint: {
    fontSize: 12,
    color: '#64748b',
    marginBottom: 18,
    marginTop: 4,
  },
  arenaBtnOuter: {
    borderRadius: 14,
    overflow: 'hidden',
    marginBottom: 20,
    shadowColor: '#bef264',
    shadowOffset: {width: 0, height: 6},
    shadowOpacity: 0.35,
    shadowRadius: 14,
    elevation: 8,
  },
  arenaGradient: {
    paddingVertical: 18,
    alignItems: 'center',
    justifyContent: 'center',
  },
  arenaText: {
    fontSize: 17,
    fontWeight: '900',
    color: '#14532d',
    letterSpacing: 1.6,
    textTransform: 'uppercase',
    fontStyle: 'italic',
  },
  footerRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    flexWrap: 'wrap',
    paddingBottom: 8,
  },
  footerMuted: {
    fontSize: 15,
    color: '#94a3b8',
  },
  footerAccent: {
    fontSize: 15,
    fontWeight: '800',
    color: '#d9f99d',
  },
});
