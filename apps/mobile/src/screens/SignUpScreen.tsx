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
  Dimensions,
} from 'react-native';
import {LinearGradient} from 'expo-linear-gradient';
import {Ionicons} from '@expo/vector-icons';
import {useRoute, useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {RouteProp} from '@react-navigation/native';
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import type {RootStackParamList} from '../types/navigation';
import {useAuth} from '../context/AuthContext';
import {register, type RegisterRequest} from '../services/api';
import {
  launchGoogleAuthBrowserFlow,
  resolvePostGoogleAuthRoute,
} from '../services/googleAuth';
import {
  dispatchAfterSignInSuccess,
  dispatchResetToMainProfileTab,
} from '../navigation/authNavigationActions';

function validatePassword(password: string): string | null {
  if (password.length < 8) {
    return 'Password must be at least 8 characters';
  }
  const hasLetter = /\p{L}/u.test(password);
  const hasNumber = /\d/.test(password);
  if (!hasLetter) return 'Password must contain at least one letter';
  if (!hasNumber) return 'Password must contain at least one number';
  return null;
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

export default function SignUpScreen() {
  const route = useRoute<RouteProp<RootStackParamList, 'SignUp'>>();
  const navigation =
    useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const insets = useSafeAreaInsets();
  const {setAuth} = useAuth();
  const returnToDebate = route.params?.returnToDebate;
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [pending, setPending] = useState<'idle' | 'email' | 'google'>('idle');
  const submitting = pending !== 'idle';
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const errors: Record<string, string> = {};
    if (!email.trim()) errors.email = 'Email is required';
    if (!password) errors.password = 'Password is required';
    else {
      const pwErr = validatePassword(password);
      if (pwErr) errors.password = pwErr;
    }
    if (!firstName.trim()) errors.first_name = 'First name is required';
    if (!lastName.trim()) errors.last_name = 'Last name is required';
    setFieldErrors(errors);
    setError(
      Object.keys(errors).length > 0 ? 'Please fix the errors below.' : null,
    );
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    setError(null);
    setFieldErrors({});
    if (!validate()) return;

    setPending('email');
    try {
      const body: RegisterRequest = {
        email: email.trim(),
        password,
        first_name: firstName.trim(),
        last_name: lastName.trim(),
      };
      const result = await register(body);

      if (result.ok) {
        await setAuth(result.data.token, result.data.user);
        dispatchAfterSignInSuccess({returnToDebate});
        return;
      }

      if (result.status === 400 && result.errors?.length) {
        const errs: Record<string, string> = {};
        result.errors.forEach(e => {
          errs[e.field] = e.message;
        });
        setFieldErrors(errs);
        setError(result.message);
      } else {
        setError(result.message || 'Sign up failed. Try again.');
      }
    } catch (_e) {
      setError('Something went wrong. Try again.');
    } finally {
      setPending('idle');
    }
  };

  const goToSignIn = () => {
    dispatchResetToMainProfileTab();
  };

  const handleGoogleSignUp = async () => {
    setError(null);
    setPending('google');
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
      const msg =
        e instanceof Error ? e.message : 'Google sign up failed. Try again.';
      setError(msg);
    } finally {
      setPending('idle');
    }
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
          {minHeight: Dimensions.get('window').height - 40},
          {paddingTop: Math.max(insets.top, 8)},
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
          <View style={styles.topBar}>
            <TouchableOpacity
              onPress={() => navigation.goBack()}
              disabled={submitting}
              hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}
              accessibilityRole="button"
              accessibilityLabel="Go back">
              <Ionicons name="chevron-back" size={28} color="#d9f99d" />
            </TouchableOpacity>
          </View>

          <View style={styles.brandRow}>
            <Ionicons name="football" size={36} color="#c7f349" />
            <Text style={styles.brandWordmark}>FUCCI</Text>
          </View>
          <Text style={styles.tagline}>
            Create your account to join the arena.
          </Text>

          {error ? <Text style={styles.errorText}>{error}</Text> : null}

          <Text style={styles.fieldLabel}>Email address</Text>
          <TextInput
            style={[styles.input, fieldErrors.email && styles.inputError]}
            placeholder="manager@fucci.fc"
            placeholderTextColor="#64748b"
            value={email}
            onChangeText={t => {
              setEmail(t);
              if (fieldErrors.email) setFieldErrors(p => ({...p, email: ''}));
            }}
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="email-address"
            editable={!submitting}
          />
          {fieldErrors.email ? (
            <Text style={styles.fieldError}>{fieldErrors.email}</Text>
          ) : null}

          <Text style={styles.fieldLabel}>Password</Text>
          <TextInput
            style={[styles.input, fieldErrors.password && styles.inputError]}
            placeholder="••••••••"
            placeholderTextColor="#64748b"
            value={password}
            onChangeText={t => {
              setPassword(t);
              if (fieldErrors.password)
                setFieldErrors(p => ({...p, password: ''}));
            }}
            secureTextEntry
            editable={!submitting}
          />
          <Text style={styles.passwordHint}>
            Min 8 characters, one letter, one number.
          </Text>
          {fieldErrors.password ? (
            <Text style={styles.fieldError}>{fieldErrors.password}</Text>
          ) : null}

          <Text style={styles.fieldLabel}>First name</Text>
          <TextInput
            style={[styles.input, fieldErrors.first_name && styles.inputError]}
            placeholder="Alex"
            placeholderTextColor="#64748b"
            value={firstName}
            onChangeText={t => {
              setFirstName(t);
              if (fieldErrors.first_name)
                setFieldErrors(p => ({...p, first_name: ''}));
            }}
            autoCapitalize="words"
            editable={!submitting}
          />
          {fieldErrors.first_name ? (
            <Text style={styles.fieldError}>{fieldErrors.first_name}</Text>
          ) : null}

          <Text style={styles.fieldLabel}>Last name</Text>
          <TextInput
            style={[styles.input, fieldErrors.last_name && styles.inputError]}
            placeholder="Morgan"
            placeholderTextColor="#64748b"
            value={lastName}
            onChangeText={t => {
              setLastName(t);
              if (fieldErrors.last_name)
                setFieldErrors(p => ({...p, last_name: ''}));
            }}
            autoCapitalize="words"
            editable={!submitting}
          />
          {fieldErrors.last_name ? (
            <Text style={styles.fieldError}>{fieldErrors.last_name}</Text>
          ) : null}

          <TouchableOpacity
            activeOpacity={0.92}
            onPress={handleSubmit}
            disabled={submitting}
            style={styles.arenaBtnOuter}>
            <LinearGradient
              colors={['#ecfccb', '#bef264', '#a3e635']}
              start={{x: 0, y: 0.5}}
              end={{x: 1, y: 0.5}}
              style={styles.arenaGradient}>
              {pending === 'email' ? (
                <ActivityIndicator color="#14532d" />
              ) : (
                <Text style={styles.arenaText}>Join the arena</Text>
              )}
            </LinearGradient>
          </TouchableOpacity>

          <View style={styles.orRow}>
            <View style={styles.orLine} />
            <Text style={styles.orText}>OR</Text>
            <View style={styles.orLine} />
          </View>

          <TouchableOpacity
            style={[styles.socialBtn, submitting && styles.disabled]}
            onPress={handleGoogleSignUp}
            disabled={submitting}
            accessibilityRole="button"
            accessibilityLabel="Continue with Google">
            <View style={styles.socialInner}>
              <Text style={styles.googleG}>G</Text>
              <Text style={styles.socialLabel}>Continue with Google</Text>
            </View>
            {pending === 'google' ? (
              <ActivityIndicator color="#e2e8f0" style={styles.socialSpinner} />
            ) : null}
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.footerRow}
            onPress={goToSignIn}
            disabled={submitting}>
            <Text style={styles.footerMuted}>Already have an account? </Text>
            <Text style={styles.footerAccent}>Sign in</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  flex: {
    flex: 1,
    backgroundColor: '#020617',
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
    paddingTop: 0,
    maxWidth: 440,
    alignSelf: 'center',
    width: '100%',
  },
  topBar: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
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
  fieldLabel: {
    fontSize: 10,
    fontWeight: '800',
    color: '#94a3b8',
    letterSpacing: 1.4,
    textTransform: 'uppercase',
    marginBottom: 8,
    marginTop: 4,
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
  inputError: {
    borderColor: '#f87171',
  },
  passwordHint: {
    fontSize: 12,
    color: '#64748b',
    marginBottom: 18,
    marginTop: 4,
  },
  fieldError: {
    color: '#fca5a5',
    fontSize: 12,
    marginBottom: 12,
  },
  arenaBtnOuter: {
    borderRadius: 14,
    overflow: 'hidden',
    marginTop: 8,
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
  socialBtn: {
    backgroundColor: '#1e293b',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#334155',
    paddingVertical: 14,
    paddingHorizontal: 16,
    marginBottom: 20,
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
