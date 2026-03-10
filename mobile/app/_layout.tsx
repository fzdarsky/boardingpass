// Polyfill crypto.getRandomValues() for secure-remote-password library.
// MUST be imported before any SRP/crypto code is loaded.
import 'react-native-get-random-values';

import { useEffect } from 'react';
import { Stack } from 'expo-router';
import { useColorScheme, StyleSheet, Text } from 'react-native';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { PaperProvider } from 'react-native-paper';
import * as SplashScreen from 'expo-splash-screen';
import Constants from 'expo-constants';
import { AppProvider } from '@/contexts/AppProvider';
import { lightTheme, darkTheme } from '@/theme';

const appVersion = Constants.expoConfig?.version ?? 'unknown';

const SPLASH_MIN_MS = 3000;

SplashScreen.preventAutoHideAsync();

export default function RootLayout() {
  const colorScheme = useColorScheme();
  const theme = colorScheme === 'dark' ? darkTheme : lightTheme;

  useEffect(() => {
    const timer = setTimeout(() => {
      SplashScreen.hideAsync();
    }, SPLASH_MIN_MS);
    return () => clearTimeout(timer);
  }, []);

  return (
    <GestureHandlerRootView style={styles.container}>
      <PaperProvider theme={theme}>
        <AppProvider>
          <Stack
            screenOptions={{
              headerStyle: {
                backgroundColor: theme.colors.primary,
              },
              headerTintColor: theme.colors.onPrimary,
              headerTitleStyle: {
                fontWeight: 'bold',
              },
            }}
          >
            <Stack.Screen
              name="index"
              options={{
                title: 'Devices',
                headerRight: () => (
                  <Text style={[styles.versionText, { color: theme.colors.onPrimary }]}>
                    v{appVersion}
                  </Text>
                ),
              }}
            />
            <Stack.Screen
              name="device/[id]"
              options={{
                title: 'Device Details',
              }}
            />
            <Stack.Screen
              name="device/authenticate"
              options={{
                title: 'Authenticate',
              }}
            />
            <Stack.Screen
              name="device/configure"
              options={{
                title: 'Configure Device',
              }}
            />
          </Stack>
        </AppProvider>
      </PaperProvider>
    </GestureHandlerRootView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  versionText: {
    fontSize: 12,
    opacity: 0.7,
    marginRight: 8,
  },
});
