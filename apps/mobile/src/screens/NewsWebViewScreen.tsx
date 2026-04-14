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
import {useSafeAreaInsets} from 'react-native-safe-area-context';
import type {RootStackParamList} from '../types/navigation';

type NewsWebViewRouteProp = RouteProp<RootStackParamList, 'NewsWebView'>;

const NewsWebViewScreen: React.FC = () => {
  const insets = useSafeAreaInsets();
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
      <WebView
        source={{uri: url}}
        style={styles.webview}
        // Default whitelist is http(s) + about:blank. Many news sites use iframes / sandbox
        // that navigate to about:srcdoc; without this, WebView tries Linking.openURL and warns.
        originWhitelist={['http://*', 'https://*', 'about:*']}
        onLoadStart={() => setIsLoading(true)}
        onLoadEnd={() => setIsLoading(false)}
        startInLoadingState={true}
        javaScriptEnabled={true}
        domStorageEnabled={true}
        allowsInlineMediaPlayback={true}
        mediaPlaybackRequiresUserAction={false}
      />

      {/* Hoisted above WebView (high z-index); native WebViews can otherwise cover siblings. */}
      <View
        style={[
          styles.closeBar,
          {
            // Sit in the same band as typical site top nav (hamburger ~56px below status bar);
            // pull up into that row so the chip covers the menu control.
            paddingTop: Math.max(insets.top - 22, 2),
            paddingLeft: Math.max(10, insets.left),
          },
        ]}
        pointerEvents="box-none">
        <TouchableOpacity
          style={[styles.closeButton, styles.closeButtonHoist]}
          onPress={handleClose}
          hitSlop={{top: 12, bottom: 12, left: 12, right: 12}}
          accessibilityRole="button"
          accessibilityLabel="Close news view">
          <Ionicons name="close" size={24} color="#1f2937" />
        </TouchableOpacity>
      </View>

      {isLoading && (
        <View style={styles.loadingContainer} pointerEvents="none">
          <ActivityIndicator size="large" color="#007AFF" />
        </View>
      )}
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
  closeBar: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'flex-start',
    alignItems: 'flex-start',
    zIndex: 100,
    elevation: 100,
  },
  closeButton: {
    padding: 8,
    borderRadius: 22,
    backgroundColor: 'rgba(255,255,255,0.94)',
    shadowColor: '#000',
    shadowOffset: {width: 0, height: 1},
    shadowOpacity: 0.2,
    shadowRadius: 3,
    elevation: 4,
  },
  /** Nudge up so the hit area overlaps fixed site headers (hamburger row). */
  closeButtonHoist: {
    marginTop: -4,
  },
  loadingContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: 'rgba(255,255,255,0.85)',
    zIndex: 50,
  },
});

export default NewsWebViewScreen;
