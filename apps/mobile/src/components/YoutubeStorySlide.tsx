import React, {useMemo} from 'react';
import {
  View,
  Text,
  StyleSheet,
  Platform,
  Linking,
  Pressable,
} from 'react-native';
import Constants from 'expo-constants';
import {WebView} from 'react-native-webview';

/**
 * YouTube’s embedded player expects a Referer that identifies the **app** (reverse-DNS id), not
 * `youtube.com` — this is a common fix for 152/153 in WebView. See:
 * https://github.com/react-native-webview/react-native-webview/discussions/3855
 * https://developers.google.com/youtube/terms/required-minimum-functionality
 */
function getAppOriginForYouTube(): string {
  const bid =
    Platform.OS === 'web'
      ? undefined
      : Platform.OS === 'ios'
        ? Constants.expoConfig?.ios?.bundleIdentifier
        : Constants.expoConfig?.android?.package;
  if (typeof bid === 'string' && bid.length > 0) {
    return `https://${bid}`;
  }
  // Fallback (same as app.json) when `expo-constants` has no config
  return 'https://com.magistridev.fucci';
}

const YT_EMBED_HOST = 'https://www.youtube-nocookie.com';

/**
 * Mobile Safari / Chrome UA. Default WebView UAs are sometimes treated as locked-down clients.
 */
const YOUTUBE_WEBVIEW_USER_AGENT = Platform.select({
  ios: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1',
  default:
    'Mozilla/5.0 (Linux; Android 14; Mobile) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36',
});

function buildEmbedRequest(videoId: string, appOrigin: string) {
  const p = new URLSearchParams({
    playsinline: '1',
    rel: '0',
    modestbranding: '1',
    autoplay: '1',
    mute: '1',
    enablejsapi: '1',
    controls: '0',
    // IFrame API: should match the embedding “site” (the app id URL).
    origin: appOrigin,
  });
  const uri = `${YT_EMBED_HOST}/embed/${encodeURIComponent(
    videoId,
  )}?${p.toString()}`;
  const headers: Record<string, string> = {
    // Critical for WKWebView: use bundle/package id, not www.youtube.com
    Referer: `${appOrigin}/`,
  };
  return {uri, headers};
}

/** Subset of YouTube `search#list` `items[]` for `youtube#video` results. */
export type YoutubeStoryResult = {
  id: {kind?: string; videoId: string};
  snippet: {channelTitle: string; title: string};
};

type YoutubeStorySlideProps = {
  result: YoutubeStoryResult;
};

export default function YoutubeStorySlide({result}: YoutubeStorySlideProps) {
  const {id, snippet} = result;
  const appOrigin = useMemo(() => getAppOriginForYouTube(), []);
  const source = useMemo(
    () => buildEmbedRequest(id.videoId, appOrigin),
    [id.videoId, appOrigin],
  );

  const openOnYouTube = () => {
    const url = `https://www.youtube.com/watch?v=${encodeURIComponent(
      id.videoId,
    )}`;
    void Linking.openURL(url);
  };

  return (
    <View style={styles.container}>
      <WebView
        source={source}
        style={styles.video}
        userAgent={YOUTUBE_WEBVIEW_USER_AGENT}
        allowsInlineMediaPlayback
        mediaPlaybackRequiresUserAction={false}
        allowsFullscreenVideo
        javaScriptEnabled
        domStorageEnabled
        sharedCookiesEnabled
        thirdPartyCookiesEnabled
        setSupportMultipleWindows={false}
      />

      <View style={styles.overlay} pointerEvents="box-none">
        <Text style={styles.channelTitle}>{snippet.channelTitle}</Text>
        <Text style={styles.title} numberOfLines={2}>
          {snippet.title}
        </Text>
        <Pressable
          onPress={openOnYouTube}
          style={styles.fallbackLink}
          hitSlop={8}
          accessibilityRole="link"
          accessibilityLabel="Open this video in the YouTube app or website">
          <Text style={styles.fallbackText}>Open in YouTube</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    width: '100%',
    backgroundColor: '#000',
  },
  video: {
    ...StyleSheet.absoluteFillObject,
  },
  overlay: {
    position: 'absolute',
    bottom: 80,
    left: 16,
    right: 16,
  },
  channelTitle: {
    color: '#aaa',
    fontSize: 13,
    fontWeight: '600',
    marginBottom: 4,
  },
  title: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '700',
  },
  fallbackLink: {
    marginTop: 10,
    alignSelf: 'flex-start',
  },
  fallbackText: {
    color: '#C6FF00',
    fontSize: 14,
    fontWeight: '600',
  },
});
