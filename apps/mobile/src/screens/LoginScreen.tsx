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
} from 'react-native';
import {useRoute, useNavigation} from '@react-navigation/native';
import type {NativeStackNavigationProp} from '@react-navigation/native-stack';
import type {RouteProp} from '@react-navigation/native';
import type {RootStackParamList} from '../types/navigation';
import {useAuth} from '../context/AuthContext';
import {getProfile, login, type LoginRequest} from '../services/api';
import {
  launchGoogleAuthBrowserFlow,
  resolvePostGoogleAuthRoute,
} from '../services/googleAuth';

export default function LoginScreen() {
  const route = useRoute<RouteProp<RootStackParamList, 'Login'>>();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const {setAuth} = useAuth();
  const returnToDebate = route.params?.returnToDebate;
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!email.trim() || !password) {
      setError('Please enter your email and password.');
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      const result = await login({email: email.trim(), password});
      if (result.ok) {
        await setAuth(result.data.token, result.data.user);
        if (returnToDebate?.match && returnToDebate?.debate) {
          navigation.reset({
            index: 0,
            routes: [{
              name: 'SingleDebate',
              params: {
                match: returnToDebate.match,
                debate: returnToDebate.debate,
                pendingAction: returnToDebate.pendingAction,
              },
            }],
          });
        } else {
          navigation.reset({
            index: 0,
            routes: [{name: 'Main'}],
          });
        }
        return;
      }
      setError(result.status === 401 ? 'Invalid credentials.' : result.message);
    } catch {
      setError('Something went wrong. Try again.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleGoogleLogin = async () => {
    setError(null);
    setSubmitting(true);
    try {
      const authResult = await launchGoogleAuthBrowserFlow();
      if (authResult.kind === 'cancel') {
        // User cancelled provider flow; intentionally no visible error.
        return;
      }
      if (authResult.kind === 'error') {
        setError(authResult.message);
        return;
      }

      const user = await getProfile(authResult.token);
      if (!user) {
        setError('Could not load your profile after Google sign-in.');
        return;
      }

      await setAuth(authResult.token, user);
      const destination = resolvePostGoogleAuthRoute(authResult.isNew);
      navigation.reset({
        index: 0,
        routes: [{name: destination}],
      });
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Google login failed. Try again.';
      setError(msg);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <KeyboardAvoidingView
      style={styles.container}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      keyboardVerticalOffset={Platform.OS === 'ios' ? 60 : 0}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        keyboardShouldPersistTaps="handled"
        showsVerticalScrollIndicator={false}>
        <Text style={styles.title}>Login</Text>
        {error ? <Text style={styles.errorText}>{error}</Text> : null}

        <TextInput
          style={styles.input}
          placeholder="Email"
          placeholderTextColor="#999"
          value={email}
          onChangeText={setEmail}
          autoCapitalize="none"
          autoCorrect={false}
          keyboardType="email-address"
          editable={!submitting}
        />
        <TextInput
          style={styles.input}
          placeholder="Password"
          placeholderTextColor="#999"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
          editable={!submitting}
        />
        <Text style={styles.passwordHint}>
          Min 8 characters, one letter, one number.
        </Text>

        <TouchableOpacity
          style={[styles.button, submitting && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={submitting}>
          {submitting ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Login</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.googleButton, submitting && styles.buttonDisabled]}
          onPress={handleGoogleLogin}
          disabled={submitting}>
          {submitting ? (
            <ActivityIndicator color="#202124" />
          ) : (
            <Text style={styles.googleButtonText}>Continue with Google</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={() => navigation.navigate('ForgotPassword')}
          disabled={submitting}>
          <Text style={styles.linkText}>Forgot password?</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={() => navigation.navigate('SignUp')}
          disabled={submitting}>
          <Text style={styles.linkText}>Don't have an account? Sign Up</Text>
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  scrollContent: {
    padding: 24,
    paddingTop: 60,
    maxWidth: 400,
    alignSelf: 'center',
    width: '100%',
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    marginBottom: 24,
    color: '#000',
  },
  errorText: {
    color: '#c00',
    marginBottom: 12,
    fontSize: 14,
  },
  input: {
    borderWidth: 1,
    borderColor: '#ddd',
    borderRadius: 10,
    paddingHorizontal: 16,
    paddingVertical: 14,
    fontSize: 16,
    marginBottom: 12,
    color: '#000',
  },
  passwordHint: {
    fontSize: 12,
    color: '#666',
    marginBottom: 16,
  },
  button: {
    backgroundColor: '#007AFF',
    borderRadius: 10,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 16,
  },
  buttonDisabled: {
    opacity: 0.7,
  },
  buttonText: {
    color: '#fff',
    fontSize: 17,
    fontWeight: '600',
  },
  googleButton: {
    borderRadius: 10,
    borderWidth: 1,
    borderColor: '#d9d9d9',
    paddingVertical: 14,
    alignItems: 'center',
    marginBottom: 16,
    backgroundColor: '#fff',
  },
  googleButtonText: {
    color: '#202124',
    fontSize: 16,
    fontWeight: '600',
  },
  linkButton: {
    alignItems: 'center',
    marginBottom: 12,
  },
  linkText: {
    color: '#007AFF',
    fontSize: 15,
  },
});
