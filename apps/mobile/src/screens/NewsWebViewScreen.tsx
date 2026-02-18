import React from 'react';
import {
  View,
  StyleSheet,
  TouchableOpacity,
  ActivityIndicator,
} from 'react-native';
import {WebView} from 'react-native-webview';
import {useRoute, useNavigation, RouteProp} from '@react-navigation/native';
import type {NavigationProp} from '@react-navigation/native';
import {Ionicons} from '@expo/vector-icons';
import type {RootStackParamList} from '../types/navigation';

type NewsWebViewRouteProp = RouteProp<RootStackParamList, 'NewsWebView'>;

const NewsWebViewScreen: React.FC = () => {
  const route = useRoute<NewsWebViewRouteProp>();
  const navigation =
    useNavigation<NavigationProp<RootStackParamList, 'NewsWebView'>>();
  const [isLoading, setIsLoading] = React.useState(true);
  const {url} = route.params;

  const handleClose = () => {
    navigation.goBack();
  };

  return (
    <View style={styles.container}>
      {isLoading && (
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color="#007AFF" />
        </View>
      )}

      <WebView
        source={{uri: url}}
        style={styles.webview}
        onLoadStart={() => setIsLoading(true)}
        onLoadEnd={() => setIsLoading(false)}
        startInLoadingState={true}
        javaScriptEnabled={true}
        domStorageEnabled={true}
        allowsInlineMediaPlayback={true}
        mediaPlaybackRequiresUserAction={false}
      />

      <View style={styles.headerSafeArea}>
        <View style={styles.header}>
          <TouchableOpacity
            style={styles.closeButton}
            onPress={handleClose}
            accessibilityLabel="Close news view">
            <Ionicons name="close" size={24} color="#333" />
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  webview: {
    flex: 1,
  },
  headerSafeArea: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'flex-end',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 8,
    backgroundColor: 'rgba(255,255,255,0.9)',
  },
  closeButton: {
    padding: 8,
    borderRadius: 20,
    backgroundColor: '#f0f0f0',
  },
  loadingContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#fff',
    zIndex: 1,
  },
});

export default NewsWebViewScreen;
