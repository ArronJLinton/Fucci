import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  Modal,
  TouchableOpacity,
} from 'react-native';

export interface AuthGateModalProps {
  visible: boolean;
  onDismiss: () => void;
  onLogin: () => void;
  onSignUp: () => void;
  title?: string;
  message?: string;
}

const DEFAULT_TITLE = 'Join the conversation';
const DEFAULT_MESSAGE =
  'Sign in or create an account to join the conversation.';

export function AuthGateModal({
  visible,
  onDismiss,
  onLogin,
  onSignUp,
  title = DEFAULT_TITLE,
  message = DEFAULT_MESSAGE,
}: AuthGateModalProps) {
  return (
    <Modal
      visible={visible}
      transparent
      animationType="fade"
      onRequestClose={onDismiss}>
      <TouchableOpacity
        activeOpacity={1}
        style={styles.backdrop}
        onPress={onDismiss}>
        <View style={styles.box}>
          <Text style={styles.title}>{title}</Text>
          <Text style={styles.message}>{message}</Text>
          <View style={styles.buttons}>
            <TouchableOpacity
              style={styles.buttonSecondary}
              onPress={onLogin}
              activeOpacity={0.8}>
              <Text style={styles.buttonSecondaryText}>Log in</Text>
            </TouchableOpacity>
            <TouchableOpacity
              style={styles.buttonPrimary}
              onPress={onSignUp}
              activeOpacity={0.8}>
              <Text style={styles.buttonPrimaryText}>Create account</Text>
            </TouchableOpacity>
          </View>
          <TouchableOpacity
            style={styles.cancel}
            onPress={onDismiss}
            activeOpacity={0.8}>
            <Text style={styles.cancelText}>Cancel</Text>
          </TouchableOpacity>
        </View>
      </TouchableOpacity>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.5)',
    justifyContent: 'center',
    padding: 24,
  },
  box: {
    backgroundColor: '#fff',
    borderRadius: 16,
    padding: 24,
  },
  title: {
    fontSize: 18,
    fontWeight: '700',
    color: '#1f2937',
    marginBottom: 8,
    textAlign: 'center',
  },
  message: {
    fontSize: 14,
    color: '#6b7280',
    textAlign: 'center',
    marginBottom: 20,
  },
  buttons: {
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'center',
    marginBottom: 12,
  },
  buttonPrimary: {
    paddingVertical: 12,
    paddingHorizontal: 20,
    backgroundColor: '#007AFF',
    borderRadius: 10,
  },
  buttonSecondary: {
    paddingVertical: 12,
    paddingHorizontal: 20,
    backgroundColor: '#e5e7eb',
    borderRadius: 10,
  },
  buttonPrimaryText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
  },
  buttonSecondaryText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#374151',
  },
  cancel: {
    alignItems: 'center',
    paddingVertical: 8,
  },
  cancelText: {
    fontSize: 14,
    color: '#6b7280',
  },
});
