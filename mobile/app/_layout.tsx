// Polyfill crypto.getRandomValues() for secure-remote-password library.
// MUST be imported before any SRP/crypto code is loaded.
import 'react-native-get-random-values';

import { useCallback, useEffect } from 'react';
import { Stack } from 'expo-router';
import { useColorScheme, StyleSheet, Text, View } from 'react-native';
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

  const renderTitle = useCallback(
    () => (
      <View style={styles.titleContainer}>
        <Text style={[styles.titleText, { color: theme.colors.onPrimary }]}>Devices</Text>
        <Text style={[styles.versionText, { color: theme.colors.onPrimary }]}>v{appVersion}</Text>
      </View>
    ),
    [theme.colors.onPrimary]
  );

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
                headerTitle: renderTitle,
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
  titleContainer: {
    flexDirection: 'row',
    alignItems: 'baseline',
    gap: 8,
  },
  titleText: {
    fontSize: 17,
    fontWeight: 'bold',
  },
  versionText: {
    fontSize: 12,
    opacity: 0.7,
  },
});
