import { useEffect } from 'react';
import { Stack } from 'expo-router';
import { useColorScheme } from 'react-native';
import { PaperProvider } from 'react-native-paper';
import * as SplashScreen from 'expo-splash-screen';
import { AppProvider } from '@/contexts/AppProvider';
import { lightTheme, darkTheme } from '@/theme';

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
              presentation: 'modal',
            }}
          />
        </Stack>
      </AppProvider>
    </PaperProvider>
  );
}
