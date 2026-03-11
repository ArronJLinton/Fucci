import React from 'react';
import {View, Text, StyleSheet, TouchableOpacity, ActivityIndicator} from 'react-native';
import {useAuth} from '../context/AuthContext';
import {rootNavigate} from '../navigation/rootNavigation';

const ProfileScreen = () => {
  const {isLoggedIn, isReady} = useAuth();

  if (!isReady) {
    return (
      <View style={[styles.container, styles.centered]}>
        <ActivityIndicator size="small" color="#007AFF" />
      </View>
    );
  }

  if (isLoggedIn) {
    return (
      <View style={styles.container}>
        <Text style={styles.text}>Profile</Text>
        <TouchableOpacity
          style={styles.button}
          onPress={() => rootNavigate('Settings')}>
          <Text style={styles.buttonText}>Settings</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <Text style={styles.text}>Profile</Text>
      <TouchableOpacity
        style={styles.button}
        onPress={() => rootNavigate('SignUp')}>
        <Text style={styles.buttonText}>Sign Up</Text>
      </TouchableOpacity>
      <TouchableOpacity
        style={styles.linkButton}
        onPress={() => rootNavigate('Login')}>
        <Text style={styles.linkText}>Log in</Text>
      </TouchableOpacity>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 24,
  },
  centered: {
    justifyContent: 'center',
  },
  text: {
    fontSize: 24,
    fontWeight: '600',
    marginBottom: 24,
  },
  button: {
    backgroundColor: '#007AFF',
    paddingVertical: 12,
    paddingHorizontal: 24,
    borderRadius: 10,
    marginBottom: 12,
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  linkButton: {
    paddingVertical: 8,
  },
  linkText: {
    color: '#007AFF',
    fontSize: 15,
  },
});

export default ProfileScreen;
