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
import {register, type RegisterRequest} from '../services/api';

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

export default function SignUpScreen() {
  const route = useRoute<RouteProp<RootStackParamList, 'SignUp'>>();
  const navigation =
    useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const {setAuth} = useAuth();
  const returnToDebate = route.params?.returnToDebate;
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [submitting, setSubmitting] = useState(false);
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

    setSubmitting(true);
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
      setSubmitting(false);
    }
  };

  const goToLogin = () => {
    navigation.navigate('Login');
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
        <Text style={styles.title}>Sign Up</Text>

        {error ? <Text style={styles.errorText}>{error}</Text> : null}

        <TextInput
          style={[styles.input, fieldErrors.email && styles.inputError]}
          placeholder="Email"
          placeholderTextColor="#999"
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

        <TextInput
          style={[styles.input, fieldErrors.password && styles.inputError]}
          placeholder="Password"
          placeholderTextColor="#999"
          value={password}
          onChangeText={t => {
            setPassword(t);
            if (fieldErrors.password)
              setFieldErrors(p => ({...p, password: ''}));
          }}
          secureTextEntry
          editable={!submitting}
        />
        <Text style={styles.passwordRules}>
          At least 8 characters, one letter, and one number.
        </Text>
        {fieldErrors.password ? (
          <Text style={styles.fieldError}>{fieldErrors.password}</Text>
        ) : null}

        <TextInput
          style={[styles.input, fieldErrors.first_name && styles.inputError]}
          placeholder="First name"
          placeholderTextColor="#999"
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

        <TextInput
          style={[styles.input, fieldErrors.last_name && styles.inputError]}
          placeholder="Last name"
          placeholderTextColor="#999"
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
          style={[styles.button, submitting && styles.buttonDisabled]}
          onPress={handleSubmit}
          disabled={submitting}>
          {submitting ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.buttonText}>Sign Up</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={goToLogin}
          disabled={submitting}>
          <Text style={styles.linkText}>Already have an account? Login</Text>
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
    marginBottom: 8,
    color: '#000',
  },
  inputError: {
    borderColor: '#c00',
  },
  passwordRules: {
    fontSize: 12,
    color: '#666',
    marginBottom: 8,
  },
  fieldError: {
    color: '#c00',
    fontSize: 12,
    marginBottom: 12,
  },
  button: {
    backgroundColor: '#007AFF',
    borderRadius: 10,
    paddingVertical: 16,
    alignItems: 'center',
    marginTop: 16,
    marginBottom: 24,
  },
  buttonDisabled: {
    opacity: 0.7,
  },
  buttonText: {
    color: '#fff',
    fontSize: 17,
    fontWeight: '600',
  },
  linkButton: {
    alignItems: 'center',
  },
  linkText: {
    color: '#007AFF',
    fontSize: 15,
  },
});
