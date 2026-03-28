/**
 * Sample React Native App
 * https://github.com/facebook/react-native
 *
 * @format
 */

import React from 'react';
import {NavigationContainer} from '@react-navigation/native';
import {rootNavigationRef} from './src/navigation/rootNavigation';
import {createBottomTabNavigator} from '@react-navigation/bottom-tabs';
import {createNativeStackNavigator} from '@react-navigation/native-stack';
import {StatusBar} from 'react-native';
import {GestureHandlerRootView} from 'react-native-gesture-handler';
import {Ionicons} from '@expo/vector-icons';
import {SafeAreaProvider, SafeAreaView} from 'react-native-safe-area-context';
import {QueryClientProvider} from '@tanstack/react-query';
import {queryClient} from './src/config/queryClient';
import {AuthProvider} from './src/context/AuthContext';

// Screens
import HomeScreen from './src/screens/HomeScreen';
import MatchDetailsScreen from './src/screens/MatchDetailsScreen';
import SingleDebateScreen from './src/screens/SingleDebateScreen';
import CameraPreviewScreen from './src/screens/CameraPreviewScreen';
import NewsWebViewScreen from './src/screens/NewsWebViewScreen';
import NewsScreen from './src/screens/NewsScreen';
import MainDebatesScreen from './src/screens/MainDebatesScreen';
import SignUpScreen from './src/screens/SignUpScreen';
import LoginScreen from './src/screens/LoginScreen';
import ForgotPasswordPlaceholderScreen from './src/screens/ForgotPasswordPlaceholderScreen';
import SettingsScreen from './src/screens/SettingsScreen';
import CreatePlayerProfileScreen from './src/screens/CreatePlayerProfileScreen';
import PlayerProfileScreen from './src/screens/PlayerProfileScreen';

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
          title: '',
          headerStyle: {
            backgroundColor: '#fff',
          },
          headerShadowVisible: false,
          headerTintColor: '#007AFF',
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

const MainStack = () => {
  return (
    <SafeAreaView style={{flex: 1, backgroundColor: '#fff'}}>
      <StatusBar barStyle="dark-content" backgroundColor="#fff" />
      <TabNavigator
        screenOptions={{
          tabBarActiveTintColor: '#007AFF',
          tabBarInactiveTintColor: '#666',
          tabBarStyle: {
            backgroundColor: '#fff',
            borderTopColor: '#e0e0e0',
            borderTopWidth: 1,
            elevation: 0,
            shadowOpacity: 0,
            height: 60,
            paddingBottom: 8,
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
            tabBarIcon: ({color, size}) => (
              <Ionicons name="football-outline" size={size} color={color} />
            ),
            tabBarLabel: () => null,
            title: '',
          }}
        />
        <Tab.Screen
          name="News"
          component={NewsStack}
          options={{
            headerShown: false,
            tabBarIcon: ({color, size}) => (
              <Ionicons name="newspaper-outline" size={size} color={color} />
            ),
            tabBarLabel: () => null,
            title: '',
          }}
        />
        <Tab.Screen
          name="Debates"
          component={DebatesStack}
          options={{
            headerShown: false,
            tabBarAccessibilityLabel: 'Debates',
            tabBarIcon: ({color, size}) => (
              <Ionicons name="chatbubbles-outline" size={size} color={color} />
            ),
            tabBarLabel: () => null,
            title: '',
          }}
        />
        <Tab.Screen
          name="Profile"
          component={SettingsScreen}
          initialParams={{embeddedInTab: true}}
          options={{
            headerShown: false,
            tabBarIcon: ({color, size}) => (
              <Ionicons name="person-outline" size={size} color={color} />
            ),
            tabBarLabel: () => null,
            title: '',
          }}
        />
      </TabNavigator>
    </SafeAreaView>
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
                  name="Login"
                  component={LoginScreen}
                  options={{title: 'Login'}}
                />
                <Stack.Screen
                  name="ForgotPassword"
                  component={ForgotPasswordPlaceholderScreen}
                  options={{title: 'Forgot password'}}
                />
                <Stack.Screen
                  name="Settings"
                  component={SettingsScreen}
                  options={{title: 'Settings'}}
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
