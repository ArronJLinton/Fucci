/**
 * Sample React Native App
 * https://github.com/facebook/react-native
 *
 * @format
 */

import React from 'react';
import {
  NavigationContainer,
  useNavigationState,
  type NavigationState,
} from '@react-navigation/native';
import {rootNavigationRef} from './src/navigation/rootNavigation';
import {
  createBottomTabNavigator,
  type BottomTabBarButtonProps,
} from '@react-navigation/bottom-tabs';
import {createNativeStackNavigator} from '@react-navigation/native-stack';
import {StatusBar} from 'react-native';
import {GestureHandlerRootView} from 'react-native-gesture-handler';
import {Ionicons} from '@expo/vector-icons';
import {SafeAreaProvider, SafeAreaView} from 'react-native-safe-area-context';
import {QueryClientProvider} from '@tanstack/react-query';
import {queryClient} from './src/config/queryClient';
import {AuthProvider} from './src/context/AuthContext';
import {
  StyledTabBarButton,
  TAB_LIME,
} from './src/navigation/StyledTabBarButton';
import {NEWS_BG} from './src/constants/newsUi';

// Screens
import HomeScreen from './src/screens/HomeScreen';
import MatchDetailsScreen from './src/screens/MatchDetailsScreen';
import SingleDebateScreen from './src/screens/SingleDebateScreen';
import CameraPreviewScreen from './src/screens/CameraPreviewScreen';
import NewsWebViewScreen from './src/screens/NewsWebViewScreen';
import NewsScreen from './src/screens/NewsScreen';
import MainDebatesScreen from './src/screens/MainDebatesScreen';
import SignUpScreen from './src/screens/SignUpScreen';
import ForgotPasswordPlaceholderScreen from './src/screens/ForgotPasswordPlaceholderScreen';
import AccountScreen from './src/screens/AccountScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import CreatePlayerProfileScreen from './src/screens/CreatePlayerProfileScreen';
import PlayerProfileScreen from './src/screens/PlayerProfileScreen';
import PlayerCompareScreen from './src/screens/PlayerCompareScreen';

// Types
import type {RootStackParamList} from './src/types/navigation';

const Tab = createBottomTabNavigator();
const Stack = createNativeStackNavigator<RootStackParamList>();

// Type assertions to fix React version compatibility
const TabNavigator = Tab.Navigator as any;
const TabScreen = Tab.Screen as any;
const StackNavigator = Stack.Navigator as any;
const StackScreen = Stack.Screen as any;
const StackGroup = Stack.Group as any;

/** Matches MainDebatesScreen / Account so safe areas and tab bar are not white bands. */
const SHELL_DEBATES_BG = '#0B0E14';
const SHELL_MATCHES_BG = '#0B0E14';
const SHELL_PROFILE_BG = '#030712';

const baseTabBarStyle = {
  borderTopWidth: 1,
  elevation: 0,
  shadowOpacity: 0,
  height: 64,
  paddingBottom: 6,
  paddingTop: 4,
} as const;

function getFocusedTabName(state?: NavigationState): string {
  if (!state) {
    return 'News';
  }
  const top = state.routes[state.index];
  if (top.name !== 'Main' || !top.state) {
    return 'Home';
  }
  const tabState = top.state as {
    index: number;
    routes: {name: string}[];
  };
  return tabState.routes[tabState.index]?.name ?? 'News';
}

const MainStack = () => {
  const focusedTab = useNavigationState(getFocusedTabName);

  const shellBg =
    focusedTab === 'Debates'
      ? SHELL_DEBATES_BG
      : focusedTab === 'Profile'
        ? SHELL_PROFILE_BG
        : focusedTab === 'News'
          ? NEWS_BG
          : focusedTab === 'Home'
            ? SHELL_MATCHES_BG
            : '#fff';

  const statusBarStyle =
    focusedTab === 'Debates' ||
    focusedTab === 'Profile' ||
    focusedTab === 'Home' ||
    focusedTab === 'News'
      ? 'light-content'
      : 'dark-content';

  const tabBarBg =
    focusedTab === 'Debates'
      ? SHELL_DEBATES_BG
      : focusedTab === 'Profile'
        ? SHELL_PROFILE_BG
        : focusedTab === 'News'
          ? NEWS_BG
          : focusedTab === 'Home'
            ? SHELL_MATCHES_BG
            : '#fff';

  const tabBarBorder =
    focusedTab === 'Debates' ||
    focusedTab === 'Profile' ||
    focusedTab === 'Home' ||
    focusedTab === 'News'
      ? 'rgba(255,255,255,0.12)'
      : '#e0e0e0';

  const inactiveTint =
    focusedTab === 'Debates' ||
    focusedTab === 'Profile' ||
    focusedTab === 'Home' ||
    focusedTab === 'News'
      ? '#8E8E93'
      : '#666';

  const tabChromeVariant =
    focusedTab === 'Debates' ||
    focusedTab === 'Profile' ||
    focusedTab === 'Home' ||
    focusedTab === 'News'
      ? 'dark'
      : 'light';

  const tabActiveTint =
    tabChromeVariant === 'dark' ? TAB_LIME : '#007AFF';

  return (
    <SafeAreaView style={{flex: 1, backgroundColor: shellBg}}>
      <StatusBar
        barStyle={statusBarStyle}
        backgroundColor={shellBg}
      />
      <TabNavigator
        initialRouteName="News"
        screenOptions={{
          tabBarActiveTintColor: tabActiveTint,
          tabBarInactiveTintColor: inactiveTint,
          tabBarShowLabel: true,
          tabBarLabelStyle: {
            fontSize: 9,
            fontWeight: '700',
            letterSpacing: 0.6,
            textTransform: 'uppercase',
            marginTop: 2,
          },
          tabBarIconStyle: {marginBottom: 0},
          tabBarButton: ({
            ref: tabRef,
            ...tabBtn
          }: BottomTabBarButtonProps) => (
            <StyledTabBarButton
              {...tabBtn}
              ref={tabRef as React.Ref<React.ComponentRef<typeof StyledTabBarButton>>}
              variant={tabChromeVariant}
            />
          ),
          tabBarStyle: {
            ...baseTabBarStyle,
            backgroundColor: tabBarBg,
            borderTopColor: tabBarBorder,
          },
          headerStyle: {
            backgroundColor: '#fff',
            elevation: 0,
            shadowOpacity: 0,
            borderBottomWidth: 1,
            borderBottomColor: '#e0e0e0',
          },
          headerTintColor: '#000',
          headerTitleStyle: {
            fontWeight: '600',
          },
        }}>
        <Tab.Screen
          name="Home"
          component={HomeStack}
          options={{
            headerShown: false,
            tabBarLabel: 'Matches',
            tabBarIcon: ({color, size, focused}) => (
              <Ionicons
                name={focused ? 'football' : 'football-outline'}
                size={size}
                color={color}
              />
            ),
            title: '',
          }}
        />
        <Tab.Screen
          name="News"
          component={NewsStack}
          options={{
            headerShown: false,
            tabBarLabel: 'News',
            tabBarIcon: ({color, size, focused}) => (
              <Ionicons
                name={focused ? 'newspaper' : 'newspaper-outline'}
                size={size}
                color={color}
              />
            ),
            title: '',
          }}
        />
        <Tab.Screen
          name="Debates"
          component={DebatesStack}
          options={{
            headerShown: false,
            tabBarAccessibilityLabel: 'Debates',
            tabBarLabel: 'Debates',
            tabBarIcon: ({color, size, focused}) => (
              <Ionicons
                name={focused ? 'chatbubbles' : 'chatbubbles-outline'}
                size={size}
                color={color}
              />
            ),
            title: '',
          }}
        />
        <Tab.Screen
          name="Profile"
          component={AccountScreen}
          initialParams={{embeddedInTab: true}}
          options={{
            headerShown: false,
            tabBarLabel: 'Profile',
            tabBarIcon: ({color, size, focused}) => (
              <Ionicons
                name={focused ? 'person' : 'person-outline'}
                size={size}
                color={color}
              />
            ),
            title: '',
          }}
        />
      </TabNavigator>
    </SafeAreaView>
  );
};

const HomeStack = () => {
  return (
    <StackNavigator>
      <StackScreen
        name="HomeTab"
        component={HomeScreen}
        options={{
          headerShown: false,
          title: '',
        }}
      />
      <StackScreen
        name="MatchDetails"
        component={MatchDetailsScreen}
        options={{
          headerShown: false,
        }}
      />
      <StackScreen
        name="NewsWebView"
        component={NewsWebViewScreen}
        options={{
          headerShown: false,
          presentation: 'card',
          animation: 'slide_from_right',
          gestureEnabled: true,
        }}
      />
      <StackScreen
        name="SingleDebate"
        component={SingleDebateScreen}
        options={{
          headerShown: false,
          presentation: 'fullScreenModal',
          animation: 'slide_from_bottom',
          gestureEnabled: true,
        }}
      />
    </StackNavigator>
  );
};

const NewsStack = () => {
  return (
    <StackNavigator>
      <StackScreen
        name="NewsFeed"
        component={NewsScreen}
        options={{
          headerShown: false,
          title: '',
        }}
      />
      <StackScreen
        name="NewsWebView"
        component={NewsWebViewScreen}
        options={{
          headerShown: false,
          presentation: 'card',
          animation: 'slide_from_right',
          gestureEnabled: true,
        }}
      />
    </StackNavigator>
  );
};

const DebatesStack = () => {
  return (
    <StackNavigator>
      <StackScreen
        name="MainDebates"
        component={MainDebatesScreen}
        options={{
          headerShown: false,
          title: '',
        }}
      />
      <StackScreen
        name="SingleDebate"
        component={SingleDebateScreen}
        options={{
          headerShown: false,
          presentation: 'fullScreenModal',
          animation: 'slide_from_bottom',
          gestureEnabled: true,
        }}
      />
      <StackScreen
        name="NewsWebView"
        component={NewsWebViewScreen}
        options={{
          headerShown: false,
          presentation: 'card',
          animation: 'slide_from_right',
          gestureEnabled: true,
        }}
      />
    </StackNavigator>
  );
};

function App(): React.JSX.Element {
  return (
    <GestureHandlerRootView style={{flex: 1}}>
      <QueryClientProvider client={queryClient}>
        <SafeAreaProvider>
          <AuthProvider>
            <NavigationContainer ref={rootNavigationRef}>
              <Stack.Navigator screenOptions={{headerShown: false}}>
                <Stack.Screen name="Main" component={MainStack} />
                <Stack.Screen
                  name="SignUp"
                  component={SignUpScreen}
                  options={{title: 'Sign Up'}}
                />
                <Stack.Screen
                  name="ForgotPassword"
                  component={ForgotPasswordPlaceholderScreen}
                  options={{title: 'Forgot password'}}
                />
                <Stack.Screen
                  name="Account"
                  component={AccountScreen}
                  options={{title: 'Account'}}
                />
                <Stack.Screen
                  name="Settings"
                  component={SettingsScreen}
                  options={{title: 'Settings', animation: 'slide_from_right'}}
                />
                <Stack.Screen
                  name="CreatePlayerProfile"
                  component={CreatePlayerProfileScreen}
                  options={{title: 'Create Player Profile'}}
                />
                <Stack.Screen
                  name="PlayerProfile"
                  component={PlayerProfileScreen}
                  options={{title: 'Player Profile'}}
                />
                <Stack.Screen
                  name="PlayerCompare"
                  component={PlayerCompareScreen}
                  options={{title: 'Compare players'}}
                />
                <Stack.Group screenOptions={{presentation: 'fullScreenModal'}}>
                  <Stack.Screen
                    name="CameraPreview"
                    component={CameraPreviewScreen}
                    options={{
                      animation: 'slide_from_bottom',
                    }}
                  />
                </Stack.Group>
              </Stack.Navigator>
            </NavigationContainer>
          </AuthProvider>
        </SafeAreaProvider>
      </QueryClientProvider>
    </GestureHandlerRootView>
  );
}

export default App;
