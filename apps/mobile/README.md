# Fucci Mobile App

React Native application built with Expo, featuring:

- Team and match information
- Debate system
- News and media
- User profiles

## Migration from Bare React Native

This app has been migrated from a bare React Native setup to Expo managed workflow for easier development.

### Key Changes Made:

- ✅ Migrated from bare React Native to Expo managed workflow
- ✅ Updated package.json with Expo dependencies
- ✅ Replaced `react-native` StatusBar with `expo-status-bar`
- ✅ Added Expo-specific configuration files (babel.config.js, metro.config.js)
- ✅ Updated app.json with proper Expo configuration
- ✅ Preserved all source code, screens, and components
- ✅ Maintained all navigation and UI functionality

### Development Commands

```bash
# Start the development server
yarn start

# Run on Android
yarn android

# Run on iOS
yarn ios

# Run on web
yarn web

# Run tests
yarn test

# Lint code
yarn lint
```

### Project Structure

```
src/
├── components/          # Reusable UI components
├── screens/            # App screens
├── services/           # API services
└── types/              # TypeScript type definitions
```

### Dependencies

- **Expo SDK 54** - Managed workflow
- **React Navigation** - Navigation library
- **React Native Paper** - Material Design components
- **React Native Vector Icons** - Icon library
- **React Native Video** - Video playback
- **React Native WebView** - Web content display

### Ejecting to Bare Workflow

If you need native functionality not available in Expo managed workflow, you can eject:

```bash
npx expo eject
```

This will create the native `android/` and `ios/` directories and allow you to add custom native code.
